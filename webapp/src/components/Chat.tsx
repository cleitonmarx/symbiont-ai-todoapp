import type { FC } from 'react';
import { ChatPanel } from '../features/chat/ChatPanel';

interface ChatProps {
  onChatDone?: () => void;
}

const Chat: FC<ChatProps> = ({ onChatDone }) => {
  return <ChatPanel onChatDone={onChatDone} />;
};

export default Chat;
