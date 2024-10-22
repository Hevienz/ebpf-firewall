import React, { useState, useEffect } from 'react'
import { date_format } from 'utils'

function Clock() {
	const [currentTime, setCurrentTime] = useState(new Date())

	useEffect(() => {
		const timer = setInterval(() => setCurrentTime(new Date()), 1000)

		return () => clearInterval(timer)
	}, [])

	return <div className="text-gray-400 text-sm">{date_format(currentTime, 'yyyy年MM月dd日 hh:mm:ss')}</div>
}

export default Clock
