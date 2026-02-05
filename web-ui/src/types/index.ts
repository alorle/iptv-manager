export interface Stream {
  acestream_id: string
  name: string
  tvg_name: string
  source: string
  enabled: boolean
  has_override: boolean
}

export interface Channel {
  name: string
  tvg_id: string
  tvg_logo: string
  group_title: string
  streams: Stream[]
}

export interface ChannelOverride {
  enabled?: boolean | null
  tvg_id?: string | null
  tvg_name?: string | null
  tvg_logo?: string | null
  group_title?: string | null
}

export interface UpdateChannelRequest {
  enabled?: boolean
  tvg_id?: string
  tvg_name?: string
  tvg_logo?: string
  group_title?: string
}
