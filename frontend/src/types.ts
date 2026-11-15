export interface Link {
  code: string
  url: string
  created_at: string
}

export interface CreateResult extends Link {
  short_url: string
}

export interface DayCount {
  day: string
  count: number
}

export interface Stats {
  code: string
  total: number
  days: DayCount[]
}
