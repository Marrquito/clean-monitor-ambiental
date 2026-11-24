import { Button } from '@/components/ui/button'
import { CHAT_SUGGESTIONS } from '@/lib/constants'

interface ChatSuggestionsProps {
  disabled: boolean
  onSelect: (question: string) => void
}

export function ChatSuggestions({ disabled, onSelect }: ChatSuggestionsProps) {
  return (
    <div className="flex gap-2 overflow-x-auto pb-1 sm:flex-wrap sm:overflow-visible" aria-label="Sugestões de perguntas">
      {CHAT_SUGGESTIONS.map(question => (
        <Button
          key={question}
          type="button"
          variant="outline"
          size="sm"
          disabled={disabled}
          onClick={() => onSelect(question)}
          className="h-auto shrink-0 rounded-full px-3 py-1.5 text-left text-xs font-normal whitespace-normal"
        >
          {question}
        </Button>
      ))}
    </div>
  )
}
