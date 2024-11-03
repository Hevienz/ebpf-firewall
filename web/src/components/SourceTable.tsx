import { Modal, PaginationProps, Table } from '@arco-design/web-react'
import { SourceStatisticResult } from 'types'
import AnimatedCounter from './AnimatedCounter'
import { getDateDiff, formatByteSizeToStr, thousandBitSeparator } from 'utils'
import { useEffect, useState, useContext, memo, useCallback } from 'react'
import * as api from '../api'
import { SorterInfo } from '@arco-design/web-react/es/Table/interface'
import { RefreshContext } from 'contexts/RefreshContext'
import { TargetTable } from './TargetTable'
import { IconEye } from '@arco-design/web-react/icon'

export const SourceTable: React.FC = memo(
	() => {
		const { pollInterval } = useContext(RefreshContext)
		const [sources, setSources] = useState<SourceStatisticResult[]>([])
		const [pagination, setPagination] = useState({
			sizeCanChange: true,
			showTotal: true,
			total: 0,
			pageSize: 20,
			current: 1,
			pageSizeChangeResetCurrent: true
		})
		const [sorting, setSorting] = useState<{
			field: string
			order: 'asc' | 'desc'
		}>({
			field: 'last_seen_at',
			order: 'desc'
		})
		const [currentSource, setCurrentSource] = useState<SourceStatisticResult | null>(null)
		const [visible, setVisible] = useState(false)

		const fetchData = async () => {
			const res = await api.getSources(pagination.current, pagination.pageSize, sorting.field, sorting.order)
			setSources(res.items)
			setPagination(prev => ({
				...prev,
				total: res.total
			}))
		}

		const onChangeTable = (pagination: PaginationProps, sorter: SorterInfo | SorterInfo[]) => {
			const { current, pageSize } = pagination
			setPagination(prev => ({ ...prev, current: current ?? 1, pageSize: pageSize ?? 20 }))
			if (!Array.isArray(sorter)) {
				sorter = [sorter]
			}
			if (sorter.length > 0) {
				const sort = sorter[0]
				console.log(sort)
				if (sort.direction) {
					setSorting({ field: sort.field as string, order: sort.direction === 'ascend' ? 'asc' : 'desc' })
					return
				}
			}
			console.log('reset')
			setSorting(prev => ({ ...prev, field: 'last_seen_at', order: 'desc' }))
		}

		useEffect(() => {
			if (pollInterval < 1) return

			fetchData()
			const timer = setInterval(fetchData, pollInterval * 1000)

			return () => clearInterval(timer)
		}, [pollInterval, pagination.current, pagination.pageSize, sorting.field, sorting.order])

		const showTargetDetails = (record: SourceStatisticResult) => {
			setCurrentSource(record)
			setVisible(true)
		}

		const hideTargetDetails = () => {
			setCurrentSource(null)
			setVisible(false)
		}

		return (
			<div>
				<Table
					data={sources}
					pagination={pagination}
					onChange={onChangeTable}
					stripe={true}
					size="small"
					scroll={{ y: 455 }}
					rowClassName={() => 'cursor-pointer'}
					rowKey={(record: any) => `${record.src_mac}-${record.src_ip}`}
					onRow={record => ({
						onClick: () => showTargetDetails(record)
					})}
					columns={[
						{
							title: '来源MAC',
							dataIndex: 'src_mac',
							ellipsis: true,
							width: 140,
							tooltip: true,
							align: 'center',
							render: value => value || '-'
						},
						{
							title: '来源IP',
							dataIndex: 'src_ip',
							ellipsis: true,
							width: 170,
							tooltip: true,
							align: 'center',
							render: value => value || '-'
						},
						{
							title: '国家',
							dataIndex: 'location.country',
							width: 80,
							tooltip: true,
							align: 'center',
							render: value => value || '-'
						},
						{
							title: '城市',
							dataIndex: 'location.city',
							width: 80,
							tooltip: true,
							align: 'center',
							render: value => value || '-'
						},
						{
							title: '包数',
							dataIndex: 'total_packets',
							width: 100,
							tooltip: true,
							align: 'center',
							sorter: true,
							render: value => (
								<AnimatedCounter
									endValue={value < 1 ? 0 : value}
									duration={pollInterval === 1 ? 500 : 1000}
									formatter={thousandBitSeparator}
								/>
							)
						},
						{
							title: '字节数',
							dataIndex: 'total_bytes',
							width: 100,
							tooltip: true,
							align: 'center',
							sorter: true,
							render: value => (
								<AnimatedCounter
									endValue={value < 1 ? 0 : value}
									duration={pollInterval === 1 ? 500 : 1000}
									formatter={formatByteSizeToStr}
								/>
							)
						},
						{
							title: '上次访问',
							dataIndex: 'last_seen_at',
							width: 120,
							tooltip: true,
							align: 'center',
							sorter: true,
							defaultSortOrder: 'descend',
							render: value => getDateDiff(value * 1000)
						}
					]}
				></Table>
				<Modal
					title={`${currentSource?.src_ip ? `${currentSource?.src_ip}（${currentSource?.src_mac}）` : currentSource?.src_mac} 来源详情`}
					visible={visible}
					okButtonProps={{ hidden: true }}
					cancelText="关闭"
					onCancel={hideTargetDetails}
					autoFocus={false}
					focusLock={true}
					style={{ width: '1024px' }}
				>
					{currentSource && <TargetTable skey={currentSource!.key} />}
				</Modal>
			</div>
		)
	},
	() => {
		return true
	}
)
