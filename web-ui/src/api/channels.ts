import type { Channel, ChannelOverride } from '../types'

const API_BASE = '/api'

// Helper to handle fetch errors with better messages
async function handleFetchError(response: Response): Promise<never> {
  let errorMessage = response.statusText

  try {
    const errorData = await response.json()
    if (errorData.message) {
      errorMessage = errorData.message
    } else if (errorData.error) {
      errorMessage =
        typeof errorData.error === 'string' ? errorData.error : JSON.stringify(errorData.error)
    }
  } catch {
    // If parsing JSON fails, use statusText
  }

  throw new Error(errorMessage)
}

// Wrapper for fetch with network error handling
async function fetchWithErrorHandling(url: string, options?: RequestInit): Promise<Response> {
  try {
    const response = await fetch(url, options)
    return response
  } catch (error) {
    // Network errors (connection refused, DNS, etc.)
    if (error instanceof TypeError) {
      throw new Error('API server is unreachable. Please check your connection.')
    }
    throw error
  }
}

export interface ListChannelsParams {
  name?: string
  group?: string
}

export async function listChannels(params?: ListChannelsParams): Promise<Channel[]> {
  const url = new URL(`${API_BASE}/channels`, window.location.origin)

  if (params?.name) {
    url.searchParams.set('name', params.name)
  }
  if (params?.group) {
    url.searchParams.set('group', params.group)
  }

  const response = await fetchWithErrorHandling(url.toString())

  if (!response.ok) {
    await handleFetchError(response)
  }

  return response.json()
}

export interface UpdateOverrideParams {
  enabled?: boolean | null
  tvg_id?: string | null
  tvg_name?: string | null
  tvg_logo?: string | null
  group_title?: string | null
}

export interface ValidationError {
  error: string
  field?: string
  message: string
  suggestions?: string[]
}

export interface ValidateResponse {
  valid: boolean
  suggestions?: string[]
}

export async function updateOverride(
  acestreamId: string,
  override: UpdateOverrideParams,
  force = false
): Promise<{ acestream_id: string; override: ChannelOverride }> {
  const url = new URL(`${API_BASE}/overrides/${acestreamId}`, window.location.origin)

  if (force) {
    url.searchParams.set('force', 'true')
  }

  const response = await fetchWithErrorHandling(url.toString(), {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(override),
  })

  if (!response.ok) {
    const errorData = await response.json().catch(() => null)
    if (errorData && errorData.error === 'validation_error') {
      throw errorData as ValidationError
    }
    await handleFetchError(response)
  }

  return response.json()
}

export async function deleteOverride(acestreamId: string): Promise<void> {
  const url = new URL(`${API_BASE}/overrides/${acestreamId}`, window.location.origin)

  const response = await fetchWithErrorHandling(url.toString(), {
    method: 'DELETE',
  })

  if (!response.ok) {
    await handleFetchError(response)
  }
}

export async function validateTvgId(tvgId: string): Promise<ValidateResponse> {
  const url = new URL(`${API_BASE}/validate/tvg-id`, window.location.origin)

  const response = await fetchWithErrorHandling(url.toString(), {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ tvg_id: tvgId }),
  })

  if (!response.ok) {
    await handleFetchError(response)
  }

  return response.json()
}

export interface BulkUpdateRequest {
  acestream_ids: string[]
  field: string
  value: string | boolean
}

export interface BulkUpdateError {
  acestream_id: string
  error: string
}

export interface BulkUpdateResponse {
  updated: number
  failed: number
  errors?: BulkUpdateError[]
}

export async function bulkUpdateOverrides(
  acestreamIds: string[],
  field: string,
  value: string | boolean,
  force = false
): Promise<BulkUpdateResponse> {
  const url = new URL(`${API_BASE}/overrides/bulk`, window.location.origin)

  if (force) {
    url.searchParams.set('force', 'true')
  }

  const response = await fetchWithErrorHandling(url.toString(), {
    method: 'PATCH',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      acestream_ids: acestreamIds,
      field,
      value,
    }),
  })

  if (!response.ok) {
    const errorData = await response.json().catch(() => null)
    if (errorData && errorData.error === 'validation_error') {
      throw errorData as ValidationError
    }
    await handleFetchError(response)
  }

  return response.json()
}
