package service

type executionRunPlan struct {
	RunLocal     bool
	RunRemote    bool
	MergeResults bool
}

func buildExecutionRunPlan(hasRemote bool, includeMaster bool) executionRunPlan {
	switch {
	case hasRemote && includeMaster:
		return executionRunPlan{RunLocal: true, RunRemote: true, MergeResults: true}
	case hasRemote:
		return executionRunPlan{RunRemote: true}
	default:
		return executionRunPlan{RunLocal: true}
	}
}
