import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import {
  Brain,
  Save,
  Loader2,
  CheckCircle2,
  Zap,
  Key,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  getLLMConfig,
  updateLLMConfig,
  testLLMConnection,
} from '@/api/llm'
import type { LLMConfig as LLMConfigType } from '@/api/llm'
import { listCredentials } from '@/api/vault'
import type { CredentialMeta } from '@/pages/vault-types'

const PROVIDERS = [
  { value: 'ollama', label: 'Ollama (Local)' },
  { value: 'openai', label: 'OpenAI' },
  { value: 'anthropic', label: 'Anthropic' },
] as const

const DEFAULT_MODELS: Record<string, string> = {
  ollama: 'qwen2.5:32b',
  openai: 'gpt-4o-mini',
  anthropic: 'claude-sonnet-4-5-20250929',
}

export function LLMConfig() {
  const queryClient = useQueryClient()

  // Local overrides -- null means "use server value"
  const [providerOverride, setProviderOverride] = useState<string | null>(null)
  const [modelOverride, setModelOverride] = useState<string | null>(null)
  const [urlOverride, setUrlOverride] = useState<string | null>(null)
  const [credentialIdOverride, setCredentialIdOverride] = useState<string | null>(null)

  const {
    data: config,
    isLoading: configLoading,
  } = useQuery({
    queryKey: ['settings', 'llm-config'],
    queryFn: getLLMConfig,
  })

  const {
    data: credentials,
    isLoading: credentialsLoading,
  } = useQuery({
    queryKey: ['vault', 'credentials'],
    queryFn: listCredentials,
  })

  // Derive current values: local override wins, then server, then defaults
  const provider = providerOverride ?? config?.provider ?? 'ollama'
  const model = modelOverride ?? config?.model ?? DEFAULT_MODELS[provider] ?? ''
  const url = urlOverride ?? config?.url ?? 'http://localhost:11434'
  const credentialId = credentialIdOverride ?? config?.credential_id ?? ''

  const apiKeyCredentials = (credentials ?? []).filter(
    (c: CredentialMeta) => c.type === 'api_key',
  )

  const needsCredential = provider === 'openai' || provider === 'anthropic'

  const saveMutation = useMutation({
    mutationFn: (cfg: LLMConfigType) => updateLLMConfig(cfg),
    onSuccess: () => {
      toast.success('LLM configuration saved')
      // Clear all local overrides -- server is now the source of truth
      setProviderOverride(null)
      setModelOverride(null)
      setUrlOverride(null)
      setCredentialIdOverride(null)
      queryClient.invalidateQueries({ queryKey: ['settings', 'llm-config'] })
    },
    onError: () => {
      toast.error('Failed to save LLM configuration')
    },
  })

  const testMutation = useMutation({
    mutationFn: testLLMConnection,
    onSuccess: (result) => {
      if (result.success) {
        toast.success(result.message || 'Connection successful')
      } else {
        toast.error(result.message || 'Connection failed')
      }
    },
    onError: () => {
      toast.error('Failed to test LLM connection')
    },
  })

  function handleProviderChange(newProvider: string) {
    setProviderOverride(newProvider)
    setModelOverride(DEFAULT_MODELS[newProvider] || '')
    setCredentialIdOverride('')
    if (newProvider === 'ollama') {
      setUrlOverride('http://localhost:11434')
    }
  }

  function handleSave() {
    const cfg: LLMConfigType = {
      provider,
      model: model || DEFAULT_MODELS[provider] || '',
    }
    if (provider === 'ollama') {
      cfg.url = url
    }
    if (needsCredential && credentialId) {
      cfg.credential_id = credentialId
    }
    saveMutation.mutate(cfg)
  }

  const isLoading = configLoading || credentialsLoading

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-medium flex items-center gap-2">
          <Brain className="h-4 w-4 text-muted-foreground" />
          LLM Provider
        </CardTitle>
        <CardDescription>
          Configure the AI model used for network analysis, anomaly
          detection, and insights. Ollama runs locally; OpenAI and
          Anthropic require an API key stored in the Vault.
        </CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
        ) : (
          <div className="space-y-4">
            {/* Provider selector */}
            <div className="space-y-2">
              <Label htmlFor="llm-provider">Provider</Label>
              <select
                id="llm-provider"
                value={provider}
                onChange={(e) => handleProviderChange(e.target.value)}
                disabled={saveMutation.isPending}
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
              >
                {PROVIDERS.map((p) => (
                  <option key={p.value} value={p.value}>
                    {p.label}
                  </option>
                ))}
              </select>
            </div>

            {/* Ollama URL */}
            {provider === 'ollama' && (
              <div className="space-y-2">
                <Label htmlFor="llm-url">Ollama URL</Label>
                <Input
                  id="llm-url"
                  value={url}
                  onChange={(e) => setUrlOverride(e.target.value)}
                  placeholder="http://localhost:11434"
                  disabled={saveMutation.isPending}
                />
              </div>
            )}

            {/* Credential selector for cloud providers */}
            {needsCredential && (
              <div className="space-y-2">
                <Label htmlFor="llm-credential">API Key Credential</Label>
                {apiKeyCredentials.length === 0 ? (
                  <p className="text-sm text-muted-foreground flex items-center gap-2">
                    <Key className="h-4 w-4" />
                    No API key credentials. Add one in Vault first.
                  </p>
                ) : (
                  <select
                    id="llm-credential"
                    value={credentialId}
                    onChange={(e) => setCredentialIdOverride(e.target.value)}
                    disabled={saveMutation.isPending}
                    className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    <option value="">Select a credential...</option>
                    {apiKeyCredentials.map((cred: CredentialMeta) => (
                      <option key={cred.id} value={cred.id}>
                        {cred.name}
                      </option>
                    ))}
                  </select>
                )}
              </div>
            )}

            {/* Model */}
            <div className="space-y-2">
              <Label htmlFor="llm-model">Model</Label>
              <Input
                id="llm-model"
                value={model}
                onChange={(e) => setModelOverride(e.target.value)}
                placeholder={DEFAULT_MODELS[provider] || 'Model name'}
                disabled={saveMutation.isPending}
              />
              <p className="text-xs text-muted-foreground">
                Default: {DEFAULT_MODELS[provider]}
              </p>
            </div>

            {/* Action buttons */}
            <div className="flex items-center gap-3 pt-2">
              <Button
                onClick={handleSave}
                disabled={saveMutation.isPending}
                className="gap-2"
              >
                {saveMutation.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : saveMutation.isSuccess ? (
                  <CheckCircle2 className="h-4 w-4" />
                ) : (
                  <Save className="h-4 w-4" />
                )}
                {saveMutation.isPending ? 'Saving...' : 'Save'}
              </Button>
              <Button
                variant="outline"
                onClick={() => testMutation.mutate()}
                disabled={testMutation.isPending || saveMutation.isPending}
                className="gap-2"
              >
                {testMutation.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : testMutation.isSuccess && testMutation.data?.success ? (
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                ) : (
                  <Zap className="h-4 w-4" />
                )}
                {testMutation.isPending ? 'Testing...' : 'Test Connection'}
              </Button>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
