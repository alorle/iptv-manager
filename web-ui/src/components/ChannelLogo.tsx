interface ChannelLogoProps {
  logo?: string
  channelName: string
}

export function ChannelLogo({ logo, channelName }: ChannelLogoProps) {
  return (
    <>
      {logo ? (
        <img
          src={logo}
          alt={`${channelName} logo`}
          className="channel-logo"
          onError={(e) => {
            e.currentTarget.style.display = 'none'
            const placeholder = e.currentTarget.nextElementSibling as HTMLElement
            if (placeholder) placeholder.style.display = 'flex'
          }}
        />
      ) : null}
      <div
        className="channel-logo-placeholder"
        style={{ display: logo ? 'none' : 'flex' }}
        aria-label="No logo available"
      >
        <svg
          width="24"
          height="24"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <rect x="2" y="7" width="20" height="15" rx="2" ry="2" />
          <polyline points="17 2 12 7 7 2" />
        </svg>
      </div>
    </>
  )
}
