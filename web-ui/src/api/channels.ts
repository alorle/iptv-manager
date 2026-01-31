import type { Channel } from '../types';

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
