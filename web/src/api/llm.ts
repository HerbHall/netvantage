import { api } from './client'

export interface LLMConfig {
  provider: string      // "ollama", "openai", "anthropic"
  model: string
  url?: string          // only for ollama
  credential_id?: string // for openai/anthropic
}

export interface LLMTestResult {
  success: boolean
  message: string
  model?: string
}

export async function getLLMConfig(): Promise<LLMConfig> {
  return api.get<LLMConfig>('/llm/config')
}

export async function updateLLMConfig(config: LLMConfig): Promise<LLMConfig> {
  return api.put<LLMConfig>('/llm/config', config)
}

export async function testLLMConnection(): Promise<LLMTestResult> {
  return api.post<LLMTestResult>('/llm/test', {})
}
