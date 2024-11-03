import { useState } from 'react'
import logo from 'assets/logo.svg'
import Dashboard from './Dashboard'
import { IconDashboard, IconUnorderedList, IconSafe } from '@arco-design/web-react/icon'
import Clock from './Clock'
import { RefreshContext } from '../contexts/RefreshContext'

function App() {
	const [selectedKey, setSelectedKey] = useState('statistics')
	const [globalPollInterval, setGlobalPollInterval] = useState(5)

	const sp = new URLSearchParams(window.location.search)
	const auth = sp.get('auth') || sp.get('token')
	if (!auth) {
		window.localStorage.clear()
		window.close()
		return
	}
	window.localStorage.setItem('auth', auth)

	const renderContent = () => {
		switch (selectedKey) {
			case 'rules':
				return <div>防护规则</div>
			case 'logs':
				return <div>防护日志</div>
			default:
				return (
					<RefreshContext.Provider value={{ pollInterval: globalPollInterval, setPollInterval: setGlobalPollInterval }}>
						<Dashboard />
					</RefreshContext.Provider>
				)
		}
	}

	const menuItems = [
		{ key: 'statistics', label: '数据统计', icon: <IconDashboard /> },
		{ key: 'rules', label: '防护规则', icon: <IconSafe />, disabled: true },
		{ key: 'logs', label: '防护日志', icon: <IconUnorderedList />, disabled: true }
	]

	return (
		<div className="flex h-screen flex-col bg-gray-100">
			<header className="h-16 bg-white border-b border-gray-200 flex items-center justify-between px-6">
				<div className="flex items-center">
					<div className="flex items-center p-4 h-16 border-b border-gray-200">
						<div className="rounded-full flex items-center justify-center mr-2">
							<img src={logo} alt="eBPF Firewall" className="w-5 h-5" />
						</div>
						<span className="text-base font-medium text-gray-800">eBPF Firewall UI</span>
					</div>
					<nav className="ml-4">
						<ul className="py-2 flex flex-row">
							{menuItems.map(item => (
								<li key={item.key} className="px-2">
									<button
										onClick={() => setSelectedKey(item.key)}
										disabled={item.disabled}
										className={`w-full text-left px-4 py-2 rounded ${
											selectedKey === item.key ? 'bg-blue-50 text-blue-600' : 'text-gray-600 hover:bg-gray-100'
										} flex items-center`}
									>
										<span className="mr-3 text-lg">{item.icon}</span>
										{item.label}
									</button>
								</li>
							))}
						</ul>
					</nav>
				</div>
				<Clock />
			</header>
			<div className="flex-1 overflow-auto p-6">{renderContent()}</div>
		</div>
	)
}

export default App
