import jp.classmethod.aws.gradle.s3.AmazonS3FileUploadTask
import java.time.LocalDateTime
import java.time.format.DateTimeFormatter
import java.io.ByteArrayOutputStream
import java.text.SimpleDateFormat
import java.util.*

buildscript {
    repositories {
        mavenLocal()
        gradlePluginPortal()
        arrayOf("public", "thirdparty", "releases").forEach { r ->
            maven {
                url = uri("${project.property("nexusBaseUrl")}/repositories/${r}")
                credentials {
                    username = project.property("nexusUserName").toString()
                    password = project.property("nexusPassword").toString()
                }
            }
        }
    }

    dependencies {
        classpath("com.xebialabs.gradle.plugins:gradle-xl-plugins-plugin:${properties["xlPluginsPluginVersion"]}")
    }
}

plugins {
    kotlin("jvm") version "1.8.10"

    id("jp.classmethod.aws") version "0.41"
    id("org.sonarqube") version "4.3.0.3225"
//    id("nebula.release") version (properties["nebulaReleasePluginVersion"] as String)
    id("maven-publish")
}

group = "com.xebialabs.xlclient"
project.defaultTasks = listOf("build")

val releasedVersion = System.getenv()["RELEASE_EXPLICIT"] ?: "23.3.0-${
    LocalDateTime.now().format(DateTimeFormatter.ofPattern("Mdd.Hmm"))
}"
project.extra.set("releasedVersion", releasedVersion)

dependencies {
    implementation(gradleApi())
    implementation(gradleKotlinDsl())

}

java {
    sourceCompatibility = JavaVersion.VERSION_11
    targetCompatibility = JavaVersion.VERSION_11
    withSourcesJar()
    withJavadocJar()
}

allprojects {
    apply(plugin = "kotlin")

    repositories {
        mavenLocal()
        mavenCentral()
        arrayOf("public", "thirdparty", "releases").forEach { r ->
            maven {
                url = uri("${project.property("nexusBaseUrl")}/repositories/${r}")
                credentials {
                    username = project.property("nexusUserName").toString()
                    password = project.property("nexusPassword").toString()
                }
            }
        }
    }
}

val goVersion = "1.19"
val packagePath = "github.com/xebialabs/blueprint-cli"
val goRootPath = "${project.rootDir}/.gogradle"
val goPath = "${goRootPath}/project_gopath"
val goCommand = "$goPath/bin/go$goVersion"
val binaryName = "xl-blueprint"
val artifactName = "blueprint-cli"
val bucketName = artifactName
val mainPath = "cmd/blueprint"

val environmentRun = mapOf(
    "GOPATH" to goPath,
)

data class Target(val os: String, val arch: String, val releaseExt: String, val ext: String = "", val upxSupported: Boolean = true) {

    fun toStringCamelCase(): String = "${os.replaceFirstChar { it.uppercaseChar() }}${arch.replaceFirstChar { it.uppercaseChar() }}"

    override fun toString(): String = "$os-$arch"

}

val targetPlatform = listOf(
    Target("darwin", "amd64", "bin", upxSupported = false), // upxSupported - removed because of Segmentation errors on the MacOS Ventura
    Target("darwin", "arm64", "bin", upxSupported = false),
    Target("linux", "amd64", "bin"),
    Target("windows", "amd64", "exe", ".exe"),
)

tasks {

    register("goPrepare") {
        group = "go"
        dependsOn("dumpVersion")
        doLast {
            exec {
                commandLine(
                    "mkdir", "-p", goPath
                )
            }
            exec {
                commandLine(
                    "go", "install", "golang.org/dl/go${goVersion}@latest"
                )
                environment(environmentRun)
            }
            exec {
                commandLine(
                    goCommand, "download"
                )
                environment(environmentRun)
            }
            val goInstalledVersion = execWithOutput {
                commandLine(
                    goCommand, "version"
                )
                environment(environmentRun)
            }
            val goPath = execWithOutput {
                commandLine(
                    goCommand, "env", "GOROOT"
                )
                environment(environmentRun)
            }

            project.logger.lifecycle("Using go version $goInstalledVersion from $goPath")
        }
    }

    register("installTemplify") {
        group = "go"
        dependsOn("goPrepare")
        doLast {
            exec {
                commandLine(
                    goCommand, "get", "github.com/wlbr/templify"
                )
                environment(environmentRun)
            }
            exec {
                commandLine(
                    goCommand, "install", "github.com/wlbr/templify"
                )
                environment(environmentRun)
            }
        }
    }

    register("installPackr") {
        group = "go"
        dependsOn("goPrepare")
        doLast {
            exec {
                commandLine(
                    goCommand, "get", "github.com/gobuffalo/packr/packr"
                )
                environment(environmentRun)
            }
            exec {
                commandLine(
                    goCommand, "install", "github.com/gobuffalo/packr/packr"
                )
                environment(environmentRun)
            }
        }
    }

    targetPlatform.forEach { target ->

        register("goBuild${target.toStringCamelCase()}") {
            group = "go"
            dependsOn("goPrepare", "installTemplify", "installPackr", "updateLicenses", "goFmt")

            doLast {
                val gitCommit = execWithOutput {
                    commandLine("git", "rev-parse", "HEAD")
                }
                val gitVersion = execWithOutput {
                    commandLine(
                        "git", "describe", "--long", "--dirty", "--always"
                    )
                }
                val gitVersionShort = if (gitVersion.startsWith("$artifactName-"))
                    gitVersion.substring(10)
                else
                    gitVersion

                val simpleDateFormat = SimpleDateFormat("yyy-MM-dd'T'HH:mm:ss.SSS'Z'")
                simpleDateFormat.setTimeZone(TimeZone.getTimeZone("UTC"))
                val date = simpleDateFormat.format(Date())

                val environmentBuild = mapOf(
                    "GOPATH" to goPath,
                    "GOARCH" to target.arch,
                    "GOOS" to target.os,
                    "GOEXE" to target.ext,
                    "CGO_ENABLED" to "0",
                )

                val params = mutableListOf(
                    goCommand,
                    "build",
                )

                val ldflags = "-ldflags=" +
                    ldFlag(
                        "CliVersion",
                        if (project.hasProperty("CliVersion") && project.property("CliVersion") != "")
                            project.property("CliVersion") as String
                        else
                            releasedVersion
                    ) +
                    ldFlag("BuildVersion", gitVersionShort) +
                    ldFlag("BuildGitCommit", gitCommit) +
                    ldFlag("BuildDate", date)
                    ldFlag("BinaryName", binaryName)
                project.logger.lifecycle("LDFlags: ${ldflags}")
                params.add(ldflags)

                if (project.hasProperty("debug")) {
                    params.add("-gcflags")
                    params.add("all=-N -l")
                }

                if (project.hasProperty("optimise")) {
                    params.add("-s")
                    params.add("-w")
                }

                exec {
                    commandLine(
                        *params.toTypedArray(),
                        "-o", "./build/${target}/$binaryName${target.ext}",
                        "-v",
                        "$mainPath/main.go",
                    )
                    environment(environmentBuild)
                }
            }
        }
    }

    register("goBuild") {
        group = "license"
        dependsOn(
            "goPrepare",
            *targetPlatform.map { "goBuild${it.toStringCamelCase()}" }.toTypedArray()
        )
    }

    register("goFmt") {
        group = "go"
        dependsOn("goPrepare")
        doLast {
            exec {
                commandLine(
                    goCommand, "fmt", "$mainPath/main.go",
                )
                environment(environmentRun)
            }
        }
    }

    targetPlatform.filter { it.upxSupported }.forEach { target ->
        register("upx${target.toStringCamelCase()}") {
            group = "go"
            dependsOn("goBuild${target.toStringCamelCase()}")
            doLast {
                exec {
                    commandLine(
                        "upx", "${project.buildDir}/$target/$binaryName${target.ext}"
                    )
                }
            }
        }
    }

    register("upx") {
        group = "go"
        dependsOn(
            *targetPlatform.map { "upx${it.toStringCamelCase()}" }.toTypedArray()
        )
    }

    register("removeLicenseFolder") {
        group = "license"
        doLast {
            delete("licenses/licences.md")
        }
    }

    task("downloadLicenses") {
        group = "license"
        dependsOn("removeLicenseFolder")
        doLast {
            val licensesDir = File("licenses")
            val goModFile = File("go.mod")
            val licensesFile = File(licensesDir, "licences.md")

            licensesDir.mkdirs()
            licensesFile.writeText("")

            val lines = goModFile.readLines()
            val regexp = Regex("^\\s+[a-zA-Z0-9.\\/\\-]+")
            val urls = lines
                .filter { regexp.matchesAt(it, 0) }
                .map { regexp.find(it, 0)?.value?.trim() }
            urls.forEach {
                licensesFile.appendText("http://$it\n")
            }
        }
    }

    register("updateLicenses") {
        group = "license"
        dependsOn("downloadLicenses")
        inputs.files("go.mod")
    }

    register("goTest") {
        group = "go"
        dependsOn("goPrepare")
        doLast {
            exec {
                commandLine(
                    goCommand, "test", "./..."
                )
                environment(environmentRun)
            }
        }
    }

    register("goClean") {
        group = "go"
        dependsOn("clean")
        doLast {
            project.delete(
                fileTree(goRootPath)
            )
        }
    }

    targetPlatform.forEach { target ->
        register<AmazonS3FileUploadTask>("upload${target.toStringCamelCase()}ToS3") {
            group = "release-dist"
            setFile(file("${project.buildDir}/$target/$binaryName${target.ext}"))
            setBucketName(bucketName)
            setKey("bin/${project.version}/${target}/$artifactName${target.ext}")
        }
    }

    register("uploadToS3") {
        group = "release-dist"
        mustRunAfter("publish")
        dependsOn(
            *targetPlatform.map { "upload${it.toStringCamelCase()}ToS3" }.toTypedArray()
        )
    }

    register("checkDependencyVersions") {
        // a placeholder to unify with release in jenkins-job
    }

    register("uploadArchives") {
        group = "upload"
        dependsOn("dumpVersion", "publish")
    }
    register("uploadArchivesMavenRepository") {
        group = "upload"
        dependsOn("dumpVersion", "publishAllPublicationsToMavenRepository")
    }
    register("uploadArchivesToMavenLocal") {
        group = "upload"
        dependsOn("dumpVersion", "publishToMavenLocal")
    }

    register("dumpVersion") {
        group = "release"
        doLast {
            file(buildDir).mkdirs()
            file("$buildDir/version.dump").writeText("version=${releasedVersion}")
        }
    }
}

tasks.withType<AbstractPublishToMaven> {
    dependsOn("build")
}

tasks.named("check") {
    dependsOn("installTemplify", "installPackr")
}


tasks.named("build") {
    dependsOn("goBuild")
}

tasks.named("test") {
    dependsOn("goTest")
}

publishing {
    publications {
        register(artifactName, MavenPublication::class) {
            targetPlatform.forEach { target ->
                artifact("${buildDir}/$target/$binaryName${target.ext}") {
                    artifactId = artifactName
                    classifier = target.toString()
                    extension = target.releaseExt
                    version = releasedVersion
                }

            }
        }
    }

    repositories {
        maven {
            val alphasRepoUrl = "/repositories/alphas/"
            val releasesRepoUrl = "/repositories/releases/"
            url = uri(
                (project.property("nexusBaseUrl") as String) +
                    (if (project.version.toString().contains("alpha")) alphasRepoUrl else releasesRepoUrl)
            )
            credentials {
                username = project.property("nexusUserName").toString()
                password = project.property("nexusPassword").toString()
            }
        }
    }
}

aws {
    profileName = "default"
}

sonarqube {
    properties {
        property("sonar.projectKey", "xl-cli")
        property("sonar.projectName", "DevOps.xl-cli")
        property("sonar.sources", "./")
        property("sonar.exclusions", "**/*_test.go,**/vendor/**")
        property("sonar.go.coverage.reportPaths", ".gogradle/reports/coverage/**/*.out")
    }
}

fun Project.execWithOutput(spec: ExecSpec.() -> Unit) = ByteArrayOutputStream().use { outputStream ->
    exec {
        this.spec()
        this.workingDir = project.rootDir
        this.standardOutput = outputStream
    }
    outputStream.toString().trim()
}

fun Project.ldFlag(c: String, v: String) = "-X \"$packagePath/$mainPath/cmd.${c}=${v}\" "

fun Project.versionLdFlag(key: String) = if (project.hasProperty(key) && project.property(key) != "")
    ldFlag(key, cleanVersions(key))
else
    ""

fun cleanVersions(key: String): String {
    val versions = project.property(key) as String
    val cleanVersions = mutableListOf<String>()
    versions.split(',').forEach {
        cleanVersions.add(it.trim())
    }
    return cleanVersions.joinToString(",")
}
