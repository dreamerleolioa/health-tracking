// BodyMetric
export interface BodyMetric {
  id: string;
  weight_kg: number | null;
  body_fat_pct: number | null;
  muscle_pct: number | null;
  visceral_fat: number | null;
  recorded_at: string;
  note: string | null;
  created_at: string;
  updated_at: string;
}

// SleepLog
export interface SleepLog {
  id: string;
  sleep_at: string;
  wake_at: string;
  duration_min: number | null;
  abnormal_wake: boolean;
  quality: number | null;
  note: string | null;
  created_at: string;
  updated_at: string;
}

// DailyActivity
export type CommuteMode = 'scooter' | 'train' | 'walk' | 'other';

export interface DailyActivity {
  id: string;
  activity_date: string;
  steps: number | null;
  commute_mode: CommuteMode | null;
  commute_minutes: number | null;
  note: string | null;
  created_at: string;
  updated_at: string;
}

// MagicPractice
export interface MagicPractice {
  id: string;
  technique_name: string;
  practiced_at: string;
  proficiency: number | null;
  duration_minutes: number | null;
  video_url: string | null;
  note: string | null;
}

export interface MagicPracticeSummary {
  technique_name: string;
  latest_proficiency: number;
  last_practiced_at: string;
  days_since_practice: number;
  needs_review: boolean;
}

// MapleSnapshot
export interface MapleSnapshot {
  id: string;
  character_name: string;
  job: string;
  level: number;
  stats: Record<string, number>;
  snapshot_at: string;
  note: string | null;
}

// API response wrappers
export interface ListResponse<T> {
  data: T[];
  meta: { total: number; from?: string; to?: string };
}

export interface ItemResponse<T> {
  data: T;
}
