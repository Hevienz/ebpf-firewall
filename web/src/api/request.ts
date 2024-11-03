import axios from 'axios'

const request = axios.create({
	baseURL: '/api/v1',
	timeout: 10000,
	withCredentials: true
})

request.interceptors.request.use(
	config => {
		const auth = localStorage.getItem('auth')
		if (auth) {
			config.headers.Authorization = auth
		}
		return config
	},
	error => Promise.reject(error)
)

request.interceptors.response.use(
	response => {
		const status = response.status
		if (status === 200) {
			return response.data
		}
		console.error(response)
		return response
	},
	error => Promise.reject(error)
)

export default request
