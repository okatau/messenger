import type { NextApiRequest, NextApiResponse } from 'next'
import { API_URL } from '../../../constants'

export default async function handlerRooms(req: NextApiRequest, res: NextApiResponse) {
    if (req.method === 'GET') {
        const response = await fetch(`${API_URL}/api/v1/rooms/`, {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': req.headers.authorization ?? '',
            },
        })
        const data = await response.json()
        return res.status(response.status).json(data)
    }

    if (req.method === 'POST') {
        const response = await fetch(`${API_URL}/api/v1/rooms/`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': req.headers.authorization ?? '',
            },
            body: JSON.stringify(req.body),
        })
        const data = await response.json()
        return res.status(response.status).json(data)
    }

    return res.status(405).end()
}
