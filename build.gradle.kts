import com.fuseanalytics.gradle.s3.S3Upload
import org.apache.commons.lang.SystemUtils.*
import org.jetbrains.kotlin.de.undercouch.gradle.tasks.download.Download
import java.io.ByteArrayOutputStream
import java.text.SimpleDateFormat
import java.time.LocalDateTime
import java.time.format.DateTimeFormatter
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

    id("com.fuseanalytics.gradle.s3") version "1.2.6"
    id("org.sonarqube") version "4.3.0.3225"
    id("nebula.release") version (properties["nebulaReleasePluginVersion"] as String)
    id("maven-publish")
}

group = "com.xebialabs.xlclient"
project.defaultTasks = listOf("build")

val releasedVersion = System.getenv()["RELEASE_EXPLICIT"] ?: "25.1.0-${
    LocalDateTime.now().format(DateTimeFormatter.ofPattern("Mdd.Hmm"))
}"
project.extra.set("releasedVersion", releasedVersion)

dependencies {
    implementation(gradleApi())
    implementation(gradleKotlinDsl())

}

java {
    sourceCompatibility = JavaVersion.VERSION_17
    targetCompatibility = JavaVersion.VERSION_17
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

var goInitialBinary = "go"
val os = detectOs()
val arch = detectHostArch()
val goVersion = "1.23.3"
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

enum class Os {
    DARWIN {
        override fun toString(): String = "darwin"
    },
    LINUX {
        override fun toString(): String = "linux"
    },
    WINDOWS {
        override fun packaging(): String = "zip"
        override fun toString(): String = "windows"
    };
    open fun packaging(): String = "tar.gz"
    fun toStringCamelCase(): String = toString().replaceFirstChar { it.uppercaseChar() }
}

enum class Arch {
    AMD64 {
        override fun toString(): String = "amd64"
    },
    ARM64 {
        override fun toString(): String = "arm64"
    };

    fun toStringCamelCase(): String = toString().replaceFirstChar { it.uppercaseChar() }
}

data class Target(val os: Os, val arch: Arch, val releaseExt: String, val ext: String = "", val upxSupported: Boolean = true) {
    fun toStringCamelCase(): String = "${os.toStringCamelCase()}${arch.toStringCamelCase()}"
    override fun toString(): String = "$os-$arch"
}

val targetPlatform = listOf(
    Target(Os.DARWIN, Arch.AMD64, "bin", upxSupported = false), // upxSupported - removed because of Segmentation errors on the MacOS Ventura
    Target(Os.DARWIN, Arch.ARM64, "bin", upxSupported = false),
    Target(Os.LINUX, Arch.AMD64, "bin"),
    Target(Os.LINUX, Arch.ARM64, "bin"),
    Target(Os.WINDOWS, Arch.AMD64, "exe", ".exe"),
)

tasks {

    register<Download>("downloadGo") {
        group = "go"
        src("https://go.dev/dl/go$goVersion.$os-$arch.${os.packaging()}")
        dest(File(goPath, "go.${os.packaging()}"))
    }

    register<Copy>("unpackGoPackage") {
        group = "go"
        dependsOn("downloadGo")
        if (os.packaging() == "tar.gz") {
            from(tarTree(File(goPath, "go.${os.packaging()}")))
        } else {
            from(zipTree(File(goPath, "go.${os.packaging()}")))
        }
        into(goPath)
    }

    register("goPrepare") {
        group = "go"
        if (project.hasProperty("useLocalGolang") && project.hasGolangInstalled()) {
            project.logger.lifecycle("Using initial go version from host")
            dependsOn("dumpVersion")
        } else {
            goInitialBinary = "$goPath/go/bin/go"
            if (File(goInitialBinary).canExecute() && project.hasGolangInstalled()) {
                project.logger.lifecycle("Using existing initial go version from project")
            } else {
                project.logger.lifecycle("Installing initial go version in project")
                dependsOn("dumpVersion", "unpackGoPackage")
            }
        }
        doLast {
            exec {
                commandLine(
                    "mkdir", "-p", goPath
                )
            }
            exec {
                commandLine(
                    goInitialBinary, "install", "golang.org/dl/go${goVersion}@latest"
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

    register("goUpdate") {
        group = "go"
        dependsOn("goPrepare")
        doLast {
            exec {
                commandLine(
                    goCommand, "get", "-u", "...",
                )
                environment(environmentRun)
            }
            exec {
                commandLine(
                    goCommand, "mod", "tidy",
                )
                environment(environmentRun)
            }
        }
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
        register<S3Upload>("upload${target.toStringCamelCase()}ToS3") {
            group = "release-dist"
            bucket = bucketName
            key = "bin/${project.version}/${target}/$binaryName${target.ext}"
            file = "${project.buildDir}/$target/$binaryName${target.ext}"
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

fun Project.hasGolangInstalled(): Boolean {
    val result = exec {
        commandLine(
            goInitialBinary, "version"
        )
        this.isIgnoreExitValue = true
        this.workingDir = project.rootDir
    }
    return result.exitValue == 0
}

fun detectOs(): Os {

    val osDetectionMap = mapOf(
        Pair(Os.LINUX, IS_OS_LINUX),
        Pair(Os.WINDOWS, IS_OS_WINDOWS),
        Pair(Os.DARWIN, IS_OS_MAC_OSX),
    )

    return osDetectionMap
        .filter { it.value }
        .firstNotNullOfOrNull { it.key } ?: throw IllegalStateException("Unrecognized os")
}

fun detectHostArch(): Arch {

    val archDetectionMap = mapOf(
        Pair("x86_64", Arch.AMD64),
        Pair("x64", Arch.AMD64),
        Pair("amd64", Arch.AMD64),
        Pair("aarch64", Arch.ARM64),
        Pair("arm64", Arch.ARM64),
    )

    val arch: String = System.getProperty("os.arch")
    if (archDetectionMap.containsKey(arch)) {
        return archDetectionMap[arch]!!
    }
    throw IllegalStateException("Unrecognized architecture: $arch")
}
