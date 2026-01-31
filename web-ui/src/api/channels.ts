import type { Channel, ChannelOverride } from '../types';

const API_BASE = '/api';

export interface ListChannelsParams {
  name?: string;
  group?: string;
}

export async function listChannels(
  params?: ListChannelsParams
): Promise<Channel[]> {
  const url = new URL(`${API_BASE}/channels`, window.location.origin);

  if (params?.name) {
    url.searchParams.set('name', params.name);
  }
  if (params?.group) {
    url.searchParams.set('group', params.group);
  }

  const response = await fetch(url.toString());

  if (!response.ok) {
    throw new Error(`Failed to fetch channels: ${response.statusText}`);
  }

  return response.json();
}

export interface UpdateOverrideParams {
  enabled?: boolean | null;
  tvg_id?: string | null;
  tvg_name?: string | null;
  tvg_logo?: string | null;
  group_title?: string | null;
}

export interface ValidationError {
  error: string;
  field?: string;
  message: string;
  suggestions?: string[];
}

export interface ValidateResponse {
  valid: boolean;
  suggestions?: string[];
}

export async function updateOverride(
  acestreamId: string,
  override: UpdateOverrideParams,
  force = false
): Promise<{ acestream_id: string; override: ChannelOverride }> {
  const url = new URL(
    `${API_BASE}/overrides/${acestreamId}`,
    window.location.origin
  );

  if (force) {
    url.searchParams.set('force', 'true');
  }

  const response = await fetch(url.toString(), {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(override),
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => null);
    if (errorData && errorData.error === 'validation_error') {
      throw errorData as ValidationError;
    }
    throw new Error(`Failed to update override: ${response.statusText}`);
  }

  return response.json();
}

export async function deleteOverride(acestreamId: string): Promise<void> {
  const url = new URL(
    `${API_BASE}/overrides/${acestreamId}`,
    window.location.origin
  );

  const response = await fetch(url.toString(), {
    method: 'DELETE',
  });

  if (!response.ok) {
    throw new Error(`Failed to delete override: ${response.statusText}`);
  }
}

export async function validateTvgId(tvgId: string): Promise<ValidateResponse> {
  const url = new URL(`${API_BASE}/validate/tvg-id`, window.location.origin);

  const response = await fetch(url.toString(), {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ tvg_id: tvgId }),
  });

  if (!response.ok) {
    throw new Error(`Failed to validate TVG-ID: ${response.statusText}`);
  }

  return response.json();
}
