import { useState, useMemo } from 'react';
import type { Channel } from '../types';
import { useChannels } from '../hooks/useChannels';
import { BulkEditModal } from './BulkEditModal';
import { bulkUpdateOverrides } from '../api/channels';
import './ChannelList.css';

interface ChannelListProps {
  onChannelSelect?: (channel: Channel) => void;
  refreshTrigger?: number;
}

export function ChannelList({ onChannelSelect, refreshTrigger }: ChannelListProps) {
  const { channels, loading, error, refetch } = useChannels(refreshTrigger);
  const [searchText, setSearchText] = useState('');
  const [groupFilter, setGroupFilter] = useState('');
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [showBulkEditModal, setShowBulkEditModal] = useState(false);
  const [bulkEditResult, setBulkEditResult] = useState<{
    type: 'success' | 'error';
    message: string;
  } | null>(null);

  // Get unique group titles for the filter dropdown
  const uniqueGroups = useMemo(() => {
    const groups = new Set(
      channels.map((ch) => ch.group_title).filter(Boolean)
    );
    return Array.from(groups).sort();
  }, [channels]);

  // Filter channels based on search and group filter
  const filteredChannels = useMemo(() => {
    return channels.filter((channel) => {
      const matchesSearch =
        searchText === '' ||
        channel.name.toLowerCase().includes(searchText.toLowerCase());

      const matchesGroup =
        groupFilter === '' || channel.group_title === groupFilter;

      return matchesSearch && matchesGroup;
    });
  }, [channels, searchText, groupFilter]);

  // Handle select all checkbox
  const handleSelectAll = (checked: boolean) => {
    if (checked) {
      setSelectedIds(
        new Set(filteredChannels.map((ch) => ch.acestream_id))
      );
    } else {
      setSelectedIds(new Set());
    }
  };

  // Handle individual checkbox
  const handleSelectOne = (id: string, checked: boolean) => {
    const newSelected = new Set(selectedIds);
    if (checked) {
      newSelected.add(id);
    } else {
      newSelected.delete(id);
    }
    setSelectedIds(newSelected);
  };

  // Handle row click
  const handleRowClick = (channel: Channel) => {
    onChannelSelect?.(channel);
  };

  // Handle bulk edit submission
  const handleBulkEdit = async (field: string, value: string | boolean) => {
    try {
      const result = await bulkUpdateOverrides(
        Array.from(selectedIds),
        field,
        value
      );

      if (result.failed > 0) {
        setBulkEditResult({
          type: 'error',
          message: `Updated ${result.updated} channel(s), but ${result.failed} failed`,
        });
      } else {
        setBulkEditResult({
          type: 'success',
          message: `Successfully updated ${result.updated} channel(s)`,
        });
      }

      // Clear selection and close modal
      setSelectedIds(new Set());
      setShowBulkEditModal(false);

      // Refresh channel list
      refetch();

      // Clear result message after 5 seconds
      setTimeout(() => setBulkEditResult(null), 5000);
    } catch (err) {
      throw err;
    }
  };

  if (loading) {
    return (
      <div className="channel-list-container">
        <div className="loading">Loading channels...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="channel-list-container">
        <div className="error">Error loading channels: {error.message}</div>
      </div>
    );
  }

  const allSelected =
    filteredChannels.length > 0 &&
    filteredChannels.every((ch) => selectedIds.has(ch.acestream_id));

  const someSelected =
    selectedIds.size > 0 &&
    !allSelected &&
    filteredChannels.some((ch) => selectedIds.has(ch.acestream_id));

  return (
    <div className="channel-list-container">
      <div className="channel-list-header">
        <h1>Channel Management</h1>
        <div className="filters">
          <input
            type="text"
            className="search-input"
            placeholder="Search channels..."
            value={searchText}
            onChange={(e) => setSearchText(e.target.value)}
          />
          <select
            className="group-filter"
            value={groupFilter}
            onChange={(e) => setGroupFilter(e.target.value)}
          >
            <option value="">All Groups</option>
            {uniqueGroups.map((group) => (
              <option key={group} value={group}>
                {group}
              </option>
            ))}
          </select>
        </div>
        {selectedIds.size > 0 && (
          <div className="bulk-actions">
            <div className="selection-info">
              {selectedIds.size} channel(s) selected
            </div>
            <button
              className="button button-primary"
              onClick={() => setShowBulkEditModal(true)}
            >
              Bulk Edit
            </button>
          </div>
        )}
        {bulkEditResult && (
          <div className={`result-message ${bulkEditResult.type}`}>
            {bulkEditResult.message}
          </div>
        )}
      </div>

      <div className="table-container">
        <table className="channel-table">
          <thead>
            <tr>
              <th className="checkbox-column">
                <input
                  type="checkbox"
                  checked={allSelected}
                  ref={(el) => {
                    if (el) el.indeterminate = someSelected;
                  }}
                  onChange={(e) => handleSelectAll(e.target.checked)}
                />
              </th>
              <th>Name</th>
              <th>Group</th>
              <th>TVG-ID</th>
              <th className="status-column">Status</th>
            </tr>
          </thead>
          <tbody>
            {filteredChannels.length === 0 ? (
              <tr>
                <td colSpan={5} className="empty-state">
                  No channels found
                </td>
              </tr>
            ) : (
              filteredChannels.map((channel) => (
                <tr
                  key={channel.acestream_id}
                  className="channel-row"
                  onClick={() => handleRowClick(channel)}
                >
                  <td
                    className="checkbox-column"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <input
                      type="checkbox"
                      checked={selectedIds.has(channel.acestream_id)}
                      onChange={(e) =>
                        handleSelectOne(channel.acestream_id, e.target.checked)
                      }
                    />
                  </td>
                  <td className="channel-name">
                    {channel.name}
                    {!channel.enabled && (
                      <span className="disabled-badge">Disabled</span>
                    )}
                  </td>
                  <td className="channel-group">{channel.group_title}</td>
                  <td className="channel-tvg-id">{channel.tvg_id || '-'}</td>
                  <td className="status-column">
                    {channel.has_override && (
                      <span
                        className="override-indicator"
                        title="Has custom overrides"
                      >
                        ⚙
                      </span>
                    )}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      <div className="channel-list-footer">
        <div className="footer-info">
          Showing {filteredChannels.length} of {channels.length} channels
        </div>
      </div>

      {showBulkEditModal && (
        <BulkEditModal
          selectedCount={selectedIds.size}
          onClose={() => setShowBulkEditModal(false)}
          onSubmit={handleBulkEdit}
        />
      )}
    </div>
  );
}
