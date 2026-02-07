import type { ReactNode } from 'react'

interface FormFieldProps {
  label: string
  htmlFor?: string
  children: ReactNode
}

export function FormField({ label, htmlFor, children }: FormFieldProps) {
  return (
    <div className="form-field">
      <label htmlFor={htmlFor}>{label}</label>
      {children}
    </div>
  )
}
