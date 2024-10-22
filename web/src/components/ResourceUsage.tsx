import { Button } from '@arco-design/web-react'
import type { SocketAction } from 'hooks/useWebSocket'
import { useState } from 'react'
import { ISystemResourceUsage } from 'types'

export default ({ socketAction }: { socketAction: SocketAction }) => {
	const [resourceUsage, setResourceUsage] = useState<ISystemResourceUsage | null>(null)

	function getResourceUsage() {
		socketAction.getSystemResourceUsage().then(r => {
			console.log(r)
			setResourceUsage(r)
		})
	}

	function formatResourceUsage(data: ISystemResourceUsage) {
		return `
主机信息:
- 主机名: ${data.host.hostname}
- 启动时间: ${new Date(data.host.boot_time * 1000).toLocaleString()}
- 操作系统: ${data.host.platform}
- 平台: ${data.host.os} v${data.host.version}

CPU:
- 名称: ${data.cpu.name}
- 物理核心数: ${data.cpu.physical_cores}、逻辑核心数: ${data.cpu.logical_cores}
- 每核心使用率: ${data.cpu.usage_per_core.map(usage => (usage * 100).toFixed(2) + '%').join(', ')}
- 总使用率: ${(data.cpu.total_usage * 100).toFixed(2)}%

内存:
- 总内存: ${(data.memory.total / 1024 / 1024 / 1024).toFixed(2)} GB、已使用: ${(data.memory.used / 1024 / 1024 / 1024).toFixed(2)} GB、可用: ${(data.memory.available / 1024 / 1024 / 1024).toFixed(2)} GB、使用率: ${data.memory.usage_rate.toFixed(2)}%

磁盘:
${data.disk
	.map(
		disk =>
			`- 挂载点: ${disk.mount_point}、设备: ${disk.device}、文件系统类型: ${disk.fs_type}、总容量: ${(disk.total / 1024 / 1024 / 1024).toFixed(2)} GB、可用空间: ${(disk.free / 1024 / 1024 / 1024).toFixed(2)} GB、已使用: ${(disk.used / 1024 / 1024 / 1024).toFixed(2)} GB、使用率: ${disk.usage_rate.toFixed(2)}%`
	)
	.join('')}

网络连接:
${data.network.connections
	.filter(x => x.status !== 'NONE')
	.map(
		conn =>
			`- 本地地址: ${conn.localaddr.ip}:${conn.localaddr.port}、远程地址: ${conn.remoteaddr.ip}:${conn.remoteaddr.port}、协议: ${conn.family === 1 ? 'IPv4' : 'IPv6'}、类型: ${conn.type === 1 ? 'TCP' : 'UDP'}、状态: ${conn.status}
`
	)
	.join('')}

网络接口:
${data.network.interfaces
	.map(
		iface =>
			`- 名称: ${iface.name}、MAC地址: ${iface.hardwareAddr}、MTU: ${iface.mtu}、标志: ${iface.flags.join(', ')}、IP地址: ${iface.addrs.map(addr => addr.addr).join(', ')}
`
	)
	.join('')}
`
	}

	return (
		<div>
			<Button onClick={getResourceUsage}>获取系统资源使用情况</Button>
			{resourceUsage && <pre style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>{formatResourceUsage(resourceUsage)}</pre>}
		</div>
	)
}
