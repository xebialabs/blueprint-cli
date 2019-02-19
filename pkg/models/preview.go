package models

type TaskPreviewItem struct {
	Name     string
	Children []TaskPreviewItem
}

type TaskInfo struct {
	Description string
	Steps       []TaskPreviewItem
}

type PreviewResponse struct {
	Task    *TaskInfo
	RawBody string
}
