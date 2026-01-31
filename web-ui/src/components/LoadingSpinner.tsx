import './LoadingSpinner.css';

interface LoadingSpinnerProps {
  size?: 'small' | 'medium' | 'large';
  inline?: boolean;
}

export function LoadingSpinner({ size = 'medium', inline = false }: LoadingSpinnerProps) {
  return (
    <div className={`loading-spinner-wrapper ${inline ? 'inline' : ''}`}>
      <div className={`loading-spinner loading-spinner-${size}`}>
        <div></div>
        <div></div>
        <div></div>
        <div></div>
      </div>
    </div>
  );
}
