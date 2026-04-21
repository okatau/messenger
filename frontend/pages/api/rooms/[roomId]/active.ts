import type { NextApiRequest, NextApiResponse } from 'next'
import { API_URL } from '../../../../constants'

export default async function handlerActiveUsers(req: NextApiRequest, res: NextApiResponse) {
    if (req.method !== 'GET') return res.status(405).end()
    console.log("active users handler")
    const { roomId } = req.query

    const response = await fetch(`${API_URL}/api/v1/rooms/${roomId}/active`, {
        method: 'GET',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': req.headers.authorization ?? '',
        },
    })

    const data = await response.json()
    return res.status(response.status).json(data)
}
