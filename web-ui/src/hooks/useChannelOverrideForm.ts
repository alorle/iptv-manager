import { useState, useCallback } from 'react'
import type { Channel } from '../types'

export interface CustomAttribute {
  key: string
  value: string
}

export interface ChannelOverrideFormState {
  enabled: boolean
  tvgId: string
  tvgName: string
  tvgLogo: string
  groupTitle: string
  customAttributes: CustomAttribute[]
  forceCheck: boolean
  setEnabled: (value: boolean) => void
  setTvgId: (value: string) => void
  setTvgName: (value: string) => void
  setTvgLogo: (value: string) => void
  setGroupTitle: (value: string) => void
  setCustomAttributes: (attributes: CustomAttribute[]) => void
  setForceCheck: (value: boolean) => void
  handleAddCustomAttribute: () => void
  handleRemoveCustomAttribute: (index: number) => void
  handleCustomAttributeChange: (index: number, field: 'key' | 'value', value: string) => void
}

/**
 * Custom hook for managing channel override form state
 * Handles all form field states and custom attribute operations
 */
export function useChannelOverrideForm(channel: Channel): ChannelOverrideFormState {
  const stream = channel.streams[0]

  const [enabled, setEnabled] = useState<boolean>(stream.enabled)
  const [tvgId, setTvgId] = useState<string>(channel.tvg_id)
  const [tvgName, setTvgName] = useState<string>(stream.tvg_name)
  const [tvgLogo, setTvgLogo] = useState<string>(channel.tvg_logo)
  const [groupTitle, setGroupTitle] = useState<string>(channel.group_title)
  const [customAttributes, setCustomAttributes] = useState<CustomAttribute[]>([])
  const [forceCheck, setForceCheck] = useState(false)

  const handleAddCustomAttribute = useCallback(() => {
    setCustomAttributes((prev) => [...prev, { key: '', value: '' }])
  }, [])

  const handleRemoveCustomAttribute = useCallback((index: number) => {
    setCustomAttributes((prev) => prev.filter((_, i) => i !== index))
  }, [])

  const handleCustomAttributeChange = useCallback(
    (index: number, field: 'key' | 'value', value: string) => {
      setCustomAttributes((prev) => {
        const updated = [...prev]
        updated[index][field] = value
        return updated
      })
    },
    []
  )

  return {
    enabled,
    tvgId,
    tvgName,
    tvgLogo,
    groupTitle,
    customAttributes,
    forceCheck,
    setEnabled,
    setTvgId,
    setTvgName,
    setTvgLogo,
    setGroupTitle,
    setCustomAttributes,
    setForceCheck,
    handleAddCustomAttribute,
    handleRemoveCustomAttribute,
    handleCustomAttributeChange,
  }
}
