import { createContext } from 'react'

export const RefreshContext = createContext<{
	pollInterval: number
	setPollInterval: (interval: number) => void
}>({
	pollInterval: 5,
	setPollInterval: () => {}
})
