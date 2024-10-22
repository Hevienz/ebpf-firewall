import { Table, Drawer, Button } from '@arco-design/web-react'
import { IInputPacket, IInputTarget, ISummary } from 'types'
import AnimatedCounter from './AnimatedCounter'
import { formatByteSizeToStr, getDateDiff, thousandBitSeparator } from 'utils'
import { IconEye } from '@arco-design/web-react/icon'
import { useEffect, useState } from 'react'

export const PacketTable: React.FC<{
	packets: IInputPacket[]
}> = ({ packets }) => {
	const [visible, setVisible] = useState(false)
	const [selectedRecord, setSelectedRecord] = useState<IInputPacket | null>(null)
	const [drawerTitle, setDrawerTitle] = useState<string>('')
	const [selectedTarget, setSelectedTarget] = useState<{
		mac: string
		ip: string
		port: [string, ISummary, number][]
		eth_type: [string, ISummary, number][]
		ip_proto: [string, ISummary, number][]
	} | null>(null)

	const showDrawer = (record: IInputPacket) => {
		setSelectedRecord(record)
		setVisible(true)
		setDrawerTitle(`来源于 ${record.src_ip ? `${record.src_ip}（${record.src_mac}）` : record.src_mac} 的访问情况`)

		const targets = Object.entries(record.target)
			.map(([_key, value]) => value)
			.sort((a, b) => b.last_time - a.last_time)
		if (targets.length > 0) {
			selectTarget(targets[0])
		} else {
			setSelectedTarget(null)
		}
	}

	const selectTarget = (record: IInputTarget) => {
		const parseSummaryItem = (data: Record<string | number, ISummary>): [string, ISummary, number][] => {
			const totalCount = Object.values(data).reduce((acc, curr) => acc + curr.count, 0)
			return Object.entries(data)
				.sort(([, a], [, b]) => b.count - a.count)
				.map(([name, data]) => [name, data, Math.max((data.count / totalCount) * 100, 1)])
		}
		setSelectedTarget({
			mac: record.mac,
			ip: record.ip,
			port: parseSummaryItem(record.port),
			eth_type: parseSummaryItem(record.eth_type),
			ip_proto: parseSummaryItem(record.ip_proto)
		})
	}

	useEffect(() => {
		if (!visible) return
		if (selectedRecord) {
			const updatedRecord = packets.find(p => p.src_mac === selectedRecord.src_mac && p.src_ip === selectedRecord.src_ip)
			if (updatedRecord) {
				setSelectedRecord(updatedRecord)
				const targets = Object.entries(updatedRecord.target)
					.map(([_key, value]) => value)
					.sort((a, b) => b.last_time - a.last_time)
				if (targets.length > 0) {
					if (selectedTarget) {
						selectTarget(targets.find(t => t.mac === selectedTarget.mac && t.ip === selectedTarget.ip) || targets[0])
					} else {
						selectTarget(targets[0])
					}
				}
			}
		}
	}, [packets, selectedRecord, visible, selectTarget])

	const renderSummary = (label: string, key: string) => {
		return (
			<div key={key} className="bg-gray-50 rounded-lg shadow-sm border border-gray-200">
				<h3 className="text-lg px-4 py-2.5 border-b border-gray-200 bg-gray-100 rounded-t-lg">{label}</h3>
				<div className="space-y-2 h-[500px] overflow-y-auto p-3">
					{(selectedTarget as any)[key]?.length ? (
						(selectedTarget as any)[key].map(([name, data, percentage]: any, index: number) => (
							<div
								key={`${key}-${name}-${index}`}
								className="flex flex-col text-sm p-3 rounded-md transition duration-300 ease-in-out hover:bg-white"
							>
								<div className="flex justify-between items-center mb-2">
									<span className="truncate flex-1">{name === '-' || !name ? '未知' : name}</span>
									<AnimatedCounter endValue={data?.count || 0} duration={1000} formatter={thousandBitSeparator} />
								</div>
								<div className="relative h-2 bg-gray-200 rounded-full overflow-hidden">
									<div
										className="absolute top-0 left-0 h-full bg-green-500 transition-all duration-300 ease-in-out"
										style={{
											width: `${percentage || 0}%`
										}}
									></div>
								</div>
							</div>
						))
					) : (
						<div className="flex justify-center items-center h-full text-gray-400 text-sm">暂无数据</div>
					)}
				</div>
			</div>
		)
	}

	return (
		<div>
			<Table
				data={packets}
				pagination={false}
				stripe={true}
				size="small"
				scroll={{ y: 495 }}
				rowKey={(record: any) => `${record.src_mac}-${record.src_ip}`}
				columns={[
					{
						title: '来源MAC',
						dataIndex: 'src_mac',
						ellipsis: true,
						width: 150,
						tooltip: true,
						align: 'center',
						render: value => value || '-'
					},
					{
						title: '来源IP',
						dataIndex: 'src_ip',
						ellipsis: true,
						width: 200,
						tooltip: true,
						align: 'center',
						render: value => value || '-'
					},
					{
						title: '国家',
						dataIndex: 'country',
						width: 80,
						tooltip: true,
						align: 'center',
						render: value => value || '-'
					},
					{
						title: '城市',
						dataIndex: 'city',
						width: 80,
						tooltip: true,
						align: 'center',
						render: value => value || '-'
					},
					{
						title: '包数',
						dataIndex: 'summary.count',
						width: 80,
						tooltip: true,
						align: 'center',
						render: value => <AnimatedCounter endValue={value} duration={1000} formatter={thousandBitSeparator} />
					},
					{
						title: '字节数',
						dataIndex: 'summary.size',
						width: 100,
						tooltip: true,
						align: 'center',
						render: value => <AnimatedCounter endValue={value} duration={1000} formatter={formatByteSizeToStr} />
					},
					{
						title: '',
						dataIndex: 'details',
						width: 60,
						align: 'center',
						render: (_, record) => (
							<div className="group" onClick={() => showDrawer(record)}>
								<IconEye
									className="group-hover:text-blue-500 transition-colors duration-200"
									style={{ cursor: 'pointer', fontSize: '18px', color: '#4E5969' }}
								/>
							</div>
						)
					}
				]}
			></Table>
			<Drawer
				width={900}
				visible={visible}
				placement="left"
				footer={null}
				title={drawerTitle}
				onOk={() => {
					setVisible(false)
				}}
				onCancel={() => {
					setVisible(false)
				}}
			>
				{selectedRecord && (
					<Table
						data={Object.entries(selectedRecord!.target)
							.map(([_key, value]) => value)
							.sort((a, b) => b.last_time - a.last_time)}
						rowKey={record => `${record.mac}-${record.ip}`}
						columns={[
							{ title: 'MAC', dataIndex: 'mac', width: 140, align: 'center' },
							{ title: 'IP', dataIndex: 'ip', width: 140, align: 'center' },
							{
								title: '包数',
								dataIndex: 'summary.count',
								width: 80,
								align: 'center',
								render: value => <AnimatedCounter endValue={value} duration={1000} formatter={thousandBitSeparator} />
							},
							{
								title: '字节数',
								dataIndex: 'summary.size',
								width: 80,
								align: 'center',
								render: value => <AnimatedCounter endValue={value} duration={1000} formatter={formatByteSizeToStr} />
							},
							{
								title: '上次访问',
								dataIndex: 'last_time',
								width: 120,
								align: 'center',
								render: value => getDateDiff(value * 1000)
							},
							{
								title: '操作',
								dataIndex: 'details',
								width: 65,
								align: 'center',
								render: (_, record) => <Button onClick={() => selectTarget(record)}>查看</Button>
							}
						]}
						pagination={false}
						size="small"
						scroll={{ y: 400 }}
					/>
				)}

				{selectedTarget && (
					<div className="mt-4" key={`${selectedTarget.mac}-${selectedTarget.ip}`}>
						<h3 className="text-lg mb-4">
							当前查看目标：{selectedTarget.ip ? `${selectedTarget.ip}（${selectedTarget.mac}）` : selectedTarget.mac}
						</h3>
						<div className="mt-4 grid grid-cols-3 gap-4">
							{renderSummary('目的端口', 'port')}
							{renderSummary('以太网类型', 'eth_type')}
							{renderSummary('协议类型', 'ip_proto')}
						</div>
					</div>
				)}
			</Drawer>
		</div>
	)
}
