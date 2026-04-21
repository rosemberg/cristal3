export interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  timestamp: Date;
}

export interface Citation {
  id: number;
  title: string;
  breadcrumb: string;
  url: string;
}

export interface ChatRequest {
  message: string;
}

export interface ChatResponse {
  response: string;
  citations?: Citation[];
}
