import React, { useContext, useEffect, useState } from 'react'
import { IconArrowUp, IconInfoCircle, IconRefresh } from '@arco-design/web-react/icon'
import ReactECharts from 'echarts-for-react'
import { thousandBitSeparator, formatByteSizeToStr, buildOption } from 'utils'
import { Alert, Dropdown, Menu } from '@arco-design/web-react'
import { Statistics, DimensionKey } from '../types'
import AnimatedCounter from './AnimatedCounter'
import { SourceTable } from './SourceTable'
import useMetrics from 'hooks/useMetrics'
import { RefreshContext } from 'contexts/RefreshContext'

const refreshIntervals = [
	{ label: '1 秒', value: 1 },
	{ label: '3 秒', value: 3 },
	{ label: '5 秒', value: 5 },
	{ label: '10 秒', value: 10 },
	{ label: '30 秒', value: 30 },
	{ label: '1 分钟', value: 60 },
	{ label: '3 分钟', value: 60 * 3 },
	{ label: '不刷新', value: 0 }
]

const Dashboard: React.FC = () => {
	const { pollInterval, setPollInterval } = useContext(RefreshContext)
	const { report, trafficDeltas, deltaMetrics, todayData } = useMetrics(pollInterval)
	const [animateDuration, setAnimateDuration] = useState(1000)

	useEffect(() => {
		setAnimateDuration(pollInterval === 1 ? 500 : 1000)
	}, [pollInterval])

	const trafficDeltaOption = buildOption(
		trafficDeltas,
		params => `${params[0].axisValue} 入站 ${thousandBitSeparator(params[0].data)} 个数据包，共 ${formatByteSizeToStr(params[1].data)}`
	)

	const thirtyDaysVisitOption = buildOption(
		report.day.map(x => ({ time: x.key, packets: x.total_packets, bytes: x.total_bytes })),
		params => `${params[0].axisValue} 入站 ${thousandBitSeparator(params[0].data)} 个数据包，共 ${formatByteSizeToStr(params[1].data)}`
	)

	const renderDimension = (x: Statistics, index: number, key: DimensionKey) => {
		const name = x.key
		const data = report.dimension[key]
		const percentage = (x.total_packets / data.reduce((acc, curr) => acc + curr.total_packets, 0)) * 100
		return (
			<div
				key={`${key}-${x.key}-${index}`}
				className="flex flex-col text-sm p-3 rounded-md transition duration-300 ease-in-out hover:bg-gray-100"
			>
				<div className="flex justify-between items-center mb-2">
					<span className="truncate flex-1">{name === '-' || !name ? '未知' : name}</span>
					<span className="ml-2">
						<AnimatedCounter endValue={x.total_packets} duration={animateDuration} formatter={thousandBitSeparator} />
					</span>
				</div>
				<div className="relative h-2 bg-gray-200 rounded-full overflow-hidden">
					<div
						className="absolute top-0 left-0 h-full bg-teal-500 transition-all duration-300 ease-in-out"
						style={{
							width: `${percentage || 0}%`
						}}
					></div>
				</div>
			</div>
		)
	}
	const renderDimensions = (label: string, key: DimensionKey) => {
		return (
			<div key={key} className="bg-white rounded-lg shadow-sm">
				<h3 className="text-lg px-4 py-2.5 border-b border-gray-200">{label}</h3>
				<div className="space-y-2 h-[250px] overflow-y-auto">
					{report.dimension[key]?.length ? (
						report.dimension[key].map((x: Statistics, index: number) => renderDimension(x, index, key))
					) : (
						<div className="flex justify-center items-center h-full text-gray-400 text-sm">暂无数据</div>
					)}
				</div>
			</div>
		)
	}

	return (
		<div className="space-y-4">
			<Alert
				type="info"
				icon={<IconInfoCircle />}
				content={
					<div className="flex justify-between items-center">
						<span>
							当前程序基于 XDP 实现，当前仅计算 <strong>入站流量</strong> 数据。
						</span>
						<Dropdown
							trigger="click"
							triggerProps={{
								showArrow: true,
								popupStyle: {
									width: '100px'
								}
							}}
							droplist={
								<Menu>
									{refreshIntervals.map(interval => (
										<Menu.Item key={interval.value.toString()} onClick={() => setPollInterval(interval.value)}>
											{interval.label}
										</Menu.Item>
									))}
								</Menu>
							}
						>
							<div className="flex items-center cursor-pointer text-gray-600 hover:text-gray-800 w-[100px]">
								<IconRefresh className="mr-1" />
								<span>刷新周期 {pollInterval}s</span>
							</div>
						</Dropdown>
					</div>
				}
			/>
			<div className="grid grid-cols-12 gap-4">
				<div className="col-span-3 flex justify-between items-center bg-white p-4 rounded-lg shadow-sm">
					<div>
						<div className="flex mb-2">
							<h3 className="text-sm text-gray-500">总包数</h3>
						</div>
						<p className="text-2xl font-semibold">
							<AnimatedCounter endValue={report.total_packets} duration={animateDuration} formatter={thousandBitSeparator} />
						</p>
					</div>
					<div>
						<div className="flex mb-2">
							<h3 className="text-sm text-gray-500">今日总包数</h3>
						</div>
						<p className="text-2xl font-semibold">
							<AnimatedCounter endValue={todayData.packets} duration={animateDuration} formatter={thousandBitSeparator} />
							<IconArrowUp className="text-green-500 w-3 h-3 ml-2" />
							<span className="text-green-500 text-sm">
								<AnimatedCounter
									endValue={deltaMetrics.packets}
									duration={animateDuration}
									formatter={thousandBitSeparator}
								/>
							</span>
						</p>
					</div>
				</div>
				<div className="col-span-3 flex justify-between items-center bg-white p-4 rounded-lg shadow-sm">
					<div>
						<div className="flex mb-2">
							<h3 className="text-sm text-gray-500">总字节数</h3>
						</div>
						<p className="text-2xl font-semibold">
							<AnimatedCounter endValue={report.total_bytes} duration={animateDuration} formatter={formatByteSizeToStr} />
						</p>
					</div>
					<div>
						<div className="flex mb-2">
							<h3 className="text-sm text-gray-500">今日总字节数</h3>
						</div>
						<p className="text-2xl font-semibold">
							<AnimatedCounter endValue={todayData.bytes} duration={animateDuration} formatter={formatByteSizeToStr} />
							<IconArrowUp className="text-green-500 w-3 h-3 ml-2" />
							<span className="text-green-500 text-sm">
								<AnimatedCounter endValue={deltaMetrics.bytes} duration={animateDuration} formatter={formatByteSizeToStr} />
							</span>
						</p>
					</div>
				</div>
				<div className="col-span-6">
					<div className="bg-white p-4 rounded-lg shadow-sm">
						<ReactECharts option={trafficDeltaOption} style={{ height: '60px', width: '100%' }} />
					</div>
				</div>
			</div>

			<div className="grid grid-cols-2 gap-4">
				<div>
					<div className="grid grid-cols-3 gap-4 mb-4">
						{[
							{ label: '访问国家', key: DimensionKey.Country },
							{ label: '访问城市', key: DimensionKey.City },
							{ label: '访问端口', key: DimensionKey.DstPort }
						].map(x => renderDimensions(x.label, x.key))}
					</div>
					<div className="grid grid-cols-3 gap-4">
						{[
							{ label: 'ETH 类型', key: DimensionKey.EthType },
							{ label: 'IP  协议', key: DimensionKey.IPProto },
							{ label: '命中规则', key: DimensionKey.Match }
						].map(x => renderDimensions(x.label, x.key))}
					</div>
				</div>
				<div>
					<div className="bg-white rounded-lg shadow-sm">
						<h3 className="text-lg px-4 py-2.5 border-b border-gray-200">来源数据</h3>
						<div className="space-y-2 h-[565px] p-4 overflow-y-auto">
							<SourceTable />
						</div>
					</div>
				</div>
			</div>

			<div className="bg-white p-4 rounded-lg shadow-sm">
				<h3 className="text-lg mb-4">30 天访问情况</h3>
				<ReactECharts option={thirtyDaysVisitOption} style={{ height: '200px', width: '100%' }} />
			</div>
		</div>
	)
}

export default Dashboard
