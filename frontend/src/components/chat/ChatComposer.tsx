import { useState, type FormEvent } from 'react'
import { SendHorizontal } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { CHAT_MAX_LENGTH } from '@/lib/constants'

interface ChatComposerProps {
  disabled: boolean
  onSubmit: (text: string) => void
}

export function ChatComposer({ disabled, onSubmit }: ChatComposerProps) {
  const [text, setText] = useState('')
  const trimmed = text.trim()

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!trimmed || disabled) {
      return
    }
    onSubmit(trimmed)
    setText('')
  }

  return (
    <form onSubmit={handleSubmit} className="flex items-center gap-2">
      <Input
        value={text}
        onChange={event => setText(event.target.value)}
        maxLength={CHAT_MAX_LENGTH}
        placeholder="Ex.: Qual foi a temperatura máxima hoje?"
        aria-label="Pergunta sobre os sensores"
        autoComplete="off"
        disabled={disabled}
        className="h-10 flex-1 text-base md:text-base"
      />
      <Button type="submit" size="icon-lg" disabled={disabled || trimmed.length === 0} aria-label="Enviar pergunta">
        <SendHorizontal aria-hidden />
      </Button>
    </form>
  )
}
