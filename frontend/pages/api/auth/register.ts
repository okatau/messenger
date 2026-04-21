import type { NextApiRequest, NextApiResponse } from 'next'
import { API_URL } from '../../../constants'

export default async function handlerRegister(req: NextApiRequest, res: NextApiResponse) {
    if (req.method !== 'POST') return res.status(405).end()

    const response = await fetch(`${API_URL}/api/v1/auth/register`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(req.body),
    })

    const data = await response.json()
    res.status(response.status).json(data)
}
