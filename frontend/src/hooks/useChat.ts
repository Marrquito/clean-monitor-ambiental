import { useCallback, useRef, useState } from 'react'
import { sendChatMessage } from '@/lib/api'

export interface ChatMessage {
  id: number
  role: 'user' | 'assistant'
  text: string
  isError: boolean
}

const UNKNOWN_ERROR_MESSAGE = 'Não foi possível obter a resposta. Tente novamente.'

export function useChat() {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [pending, setPending] = useState(false)
  const nextId = useRef(0)

  const append = useCallback((message: Omit<ChatMessage, 'id'>) => {
    nextId.current += 1
    const id = nextId.current
    setMessages(previous => [...previous, { ...message, id }])
  }, [])

  const send = useCallback(
    async (rawText: string) => {
      const text = rawText.trim()
      if (!text || pending) {
        return
      }

      append({ role: 'user', text, isError: false })
      setPending(true)
      try {
        const response = await sendChatMessage(text)
        append({ role: 'assistant', text: response.answer, isError: false })
      } catch (error) {
        const message = error instanceof Error && error.message ? error.message : UNKNOWN_ERROR_MESSAGE
        append({ role: 'assistant', text: message, isError: true })
      } finally {
        setPending(false)
      }
    },
    [append, pending],
  )

  return { messages, pending, send }
}
