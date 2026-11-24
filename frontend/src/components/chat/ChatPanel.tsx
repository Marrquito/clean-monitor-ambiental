import { useEffect, useRef } from 'react'
import { Loader2, MessageSquareText } from 'lucide-react'
import { ChatComposer } from '@/components/chat/ChatComposer'
import { ChatMessageBubble } from '@/components/chat/ChatMessageBubble'
import { ChatSuggestions } from '@/components/chat/ChatSuggestions'
import { StatusPill } from '@/components/common/StatusPill'
import { Card, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { useChat } from '@/hooks/useChat'

interface ChatPanelProps {
  aiAvailable: boolean | null
}

export function ChatPanel({ aiAvailable }: ChatPanelProps) {
  const { messages, pending, send } = useChat()
  const logRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const log = logRef.current
    if (log) {
      log.scrollTop = log.scrollHeight
    }
  }, [messages.length, pending])

  return (
    <Card className="gap-0 overflow-hidden py-0 lg:sticky lg:top-6 h-[72dvh] min-h-[480px] lg:h-[calc(100dvh-3rem)]">
      <CardHeader className="border-b border-border px-4 py-4">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <CardTitle className="flex items-center gap-2 text-base">
              <MessageSquareText className="size-4 text-primary" aria-hidden />
              Pergunte aos sensores
            </CardTitle>
            <CardDescription className="mt-1">Consulte seus sensores em linguagem natural</CardDescription>
          </div>
          {aiAvailable !== null && (
            <StatusPill tone={aiAvailable ? 'good' : 'warning'} icon={MessageSquareText}>
              {aiAvailable ? 'IA disponível' : 'IA indisponível'}
            </StatusPill>
          )}
        </div>
      </CardHeader>

      <div ref={logRef} role="log" aria-live="polite" className="flex-1 space-y-3 overflow-y-auto p-4">
        {messages.length === 0 && (
          <p className="py-6 text-center text-sm text-muted-foreground">
            Faça uma pergunta sobre temperatura, umidade ou luminosidade. As respostas usam apenas as medições
            registradas.
          </p>
        )}
        {messages.map(message => (
          <ChatMessageBubble key={message.id} message={message} />
        ))}
        {pending && (
          <div
            role="status"
            className="mr-auto flex items-center gap-2 rounded-2xl rounded-bl-md bg-secondary px-3.5 py-2.5 text-sm text-muted-foreground"
          >
            <Loader2 className="size-4 animate-spin" aria-hidden />
            Consultando as medições…
          </div>
        )}
      </div>

      <div className="space-y-3 border-t border-border p-4">
        <ChatSuggestions disabled={pending} onSelect={send} />
        <ChatComposer disabled={pending} onSubmit={send} />
      </div>
    </Card>
  )
}
