import { useState } from 'react'
import useWebSocket from 'hooks/useWebSocket'
import logo from 'assets/logo.svg'
import Dashboard from './Dashboard'
import ResourceUsage from './ResourceUsage'
import { IconDashboard, IconUnorderedList, IconSafe, IconSettings } from '@arco-design/web-react/icon'
import Clock from './Clock'

function App() {
	const host = window.location.host
	const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
	const { summaryData, incPacketHistory, dayHistory, socketAction } = useWebSocket(`${protocol}://${host}/ws`)
	const [selectedKey, setSelectedKey] = useState('1')

	const renderContent = () => {
		switch (selectedKey) {
			case '2':
				return <ResourceUsage socketAction={socketAction} />
			case '3':
				return <div>防护配置</div>
			case '4':
				return <div>防护日志</div>
			case '5':
				return <div>系统设置</div>
			default:
				return <Dashboard summaryData={summaryData} incPacketHistory={incPacketHistory} dayHistory={dayHistory} />
		}
	}

	const menuItems = [
		{ key: '1', label: '数据统计', icon: <IconDashboard /> },
		{ key: '2', label: '系统资源', icon: <IconDashboard />, disabled: true },
		{ key: '3', label: '监听规则&黑名单', icon: <IconSafe />, disabled: true },
		{ key: '4', label: '防护日志', icon: <IconUnorderedList />, disabled: true },
		{ key: '5', label: '系统设置', icon: <IconSettings />, disabled: true }
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
