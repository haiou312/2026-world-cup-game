export interface Participant {
  id: number
  name: string
  assigned: boolean
  team_id?: number
  team_name?: string
  team_code?: string | null
  flag_url?: string | null
  group_label?: string | null
  round?: number
  furthest_stage?: string
  eliminated?: boolean
  champion?: boolean
}

export interface ParticipantsResp {
  total: number
  assigned: number
  unassigned: number
  remaining: number
  participants: Participant[]
}

export interface Assignment {
  participant_id: number
  team_id: number
  team_name: string
  team_code: string | null
  flag_url: string | null
  group_label: string | null
  round: number
}

export interface TeamSide {
  team_id: number | null
  name: string | null
  code?: string | null
  flag_url?: string | null
  players?: string[]
  eliminated?: boolean
  furthest_stage?: string
  champion?: boolean
  tbd?: boolean
  seed?: string // e.g. "1E", "3 ABCDF" — shown before the team is decided
}

export interface Fixture {
  id: number
  stage: string
  status: string
  round_label: string | null
  group_label: string | null
  kickoff_at: string | null
  home_score: number | null
  away_score: number | null
  winner_team_id: number | null
  home: TeamSide
  away: TeamSide
}

export interface Round {
  stage: string
  fixtures: Fixture[]
}

export type AdvStatus = 'advancing' | 'third' | 'out' | 'pending'

export interface GroupTeam {
  team_id: number
  name: string
  code: string | null
  flag_url: string | null
  group_label: string | null
  players: string[]
  eliminated?: boolean
  furthest_stage?: string
  champion?: boolean
  // standings (present once synced)
  position?: number
  played?: number
  won?: number
  draw?: number
  lost?: number
  goals_for?: number
  goals_against?: number
  goal_diff?: number
  points?: number
  adv_status?: AdvStatus
  third_qualifying?: boolean
  third_decided?: boolean
}

export interface Group {
  group: string
  teams: GroupTeam[]
}

export interface Bracket {
  groups: Group[]
  rounds: Round[]
  champion: GroupTeam | null
}

export interface SyncStatus {
  api_key_configured: boolean
  api_key_masked: string
  last_synced_at: string | null
  last_success_at: string | null
  last_error: string
  last_error_at: string | null
  last_warnings: string
  today_done: boolean
  api_calls_today: number
  calls_date: string | null
}
