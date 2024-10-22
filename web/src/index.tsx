import { createRoot } from 'react-dom/client'
import 'tailwindcss/tailwind.css'
import App from 'components/App'
import './index.css'

createRoot(document.getElementById('root') as HTMLDivElement).render(<App />)
