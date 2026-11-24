import { TriangleAlert } from 'lucide-react'
import type { ChatMessage } from '@/hooks/useChat'
import { cn } from '@/lib/utils'

export function ChatMessageBubble({ message }: { message: ChatMessage }) {
  if (message.isError) {
    return (
      <div
        role="alert"
        className="flex max-w-[92%] items-start gap-2 rounded-2xl rounded-bl-md border border-status-critical/40 bg-status-critical/10 px-3.5 py-2.5 text-sm"
      >
        <TriangleAlert className="mt-0.5 size-4 shrink-0 text-status-critical" aria-hidden />
        <p className="break-words">{message.text}</p>
      </div>
    )
  }

  const isUser = message.role === 'user'

  return (
    <div
      className={cn(
        'max-w-[88%] rounded-2xl px-3.5 py-2.5 text-sm leading-relaxed break-words whitespace-pre-wrap',
        isUser
          ? 'ml-auto rounded-br-md bg-primary text-primary-foreground'
          : 'mr-auto rounded-bl-md bg-secondary text-secondary-foreground',
      )}
    >
      {message.text}
    </div>
  )
}
