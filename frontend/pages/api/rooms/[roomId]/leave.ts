import type { NextApiRequest, NextApiResponse } from 'next'
import { API_URL } from '../../../../constants'

export default async function handlerLeaveRoom(req: NextApiRequest, res: NextApiResponse) {
    if (req.method !== 'POST') return res.status(405).end()

    const { roomId } = req.query

    const response = await fetch(`${API_URL}/api/v1/rooms/${roomId}/leave`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': req.headers.authorization ?? '',
        },
    })

    return res.status(response.status).end()
}
