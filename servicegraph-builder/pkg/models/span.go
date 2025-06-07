package models

type Span struct {
	OperationName string
	Attributes    map[string]string
}

type K8sMetadata struct {
	Namespace string
	OwnerKind string
	OwnerName string
	OwnerUID  string
}

type EnrichedSpan struct {
	Span
	ServiceName   string
	HashableName  string
	CallerService string
	CalleeService string
	K8sMetadata   K8sMetadata
}
