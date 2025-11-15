package entity

type Statistics struct {
	TotalPRs          int            `json:"total_prs"`
	OpenPRs           int            `json:"open_prs"`
	MergedPRs         int            `json:"merged_prs"`
	AssignmentsByUser map[string]int `json:"assignments_by_user"`
	AssignmentsByPR   map[string]int `json:"assignments_by_pr"`
	TotalTeams        int            `json:"total_teams"`
	TotalUsers        int            `json:"total_users"`
	ActiveUsers       int            `json:"active_users"`
}
