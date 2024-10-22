import { useEffect, useRef, useState } from 'react'

// 新增的动画单元格组件
const AnimatedCounter: React.FC<{
	endValue: number
	duration: number
	formatter: (value: number) => string
}> = ({ endValue, duration, formatter }) => {
	const [count, setCount] = useState(endValue)
	const [isAnimating, setIsAnimating] = useState(false)
	const startValueRef = useRef(endValue)
	const startTimeRef = useRef(0)

	useEffect(() => {
		if (endValue !== startValueRef.current) {
			setIsAnimating(true)
			startValueRef.current = count
			startTimeRef.current = Date.now()

			const animateCount = () => {
				const now = Date.now()
				const progress = Math.min((now - startTimeRef.current) / duration, 1)
				const currentCount = Math.floor(startValueRef.current + progress * (endValue - startValueRef.current))

				setCount(currentCount)

				if (progress < 1) {
					requestAnimationFrame(animateCount)
				} else {
					setCount(endValue)
					setTimeout(() => setIsAnimating(false), 500) // 保持绿色效果0.5秒
				}
			}

			requestAnimationFrame(animateCount)
		}
	}, [endValue, duration])

	return <span className={`transition-colors duration-500 ${isAnimating ? 'text-green-500' : ''}`}>{formatter(count)}</span>
}

export default AnimatedCounter
