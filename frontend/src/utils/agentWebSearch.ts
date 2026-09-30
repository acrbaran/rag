import type { WebSearchProviderEntity } from '@/api/web-search-provider';

export type AgentWebSearchConfig = {
  web_search_enabled?: boolean;
  web_search_provider_id?: string;
};

/** Aracının gerçekten kullanacağı arama motoru ID'sini çözümle (arka uçtaki agent > tenant default mantığıyla tutarlı)*/
export function resolveAgentWebSearchProviderId(
  config: AgentWebSearchConfig | undefined,
  providers: WebSearchProviderEntity[],
): string | null {
  const explicitId = config?.web_search_provider_id?.trim();
  if (explicitId) {
    return providers.some((p) => p.id === explicitId) ? explicitId : null;
  }
  const defaultProvider = providers.find((p) => p.is_default);
  return defaultProvider?.id ?? null;
}

export function isAgentWebSearchEnabled(config: AgentWebSearchConfig | undefined): boolean {
  return config?.web_search_enabled === true;
}

/** Aracının web araması etkin ve kullanılabilir bir arama motoru çözümlenebiliyor*/
export function isAgentWebSearchReady(
  config: AgentWebSearchConfig | undefined,
  providers: WebSearchProviderEntity[],
  sourceWorkspaceReady?: boolean,
): boolean {
  if (!isAgentWebSearchEnabled(config)) return false;
  if (sourceWorkspaceReady !== undefined) return sourceWorkspaceReady;
  return resolveAgentWebSearchProviderId(config, providers) !== null;
}

/** Alan düzeyi varsayılan arama motorunun kullanılabilir olup olmadığı (aracı kısıtlaması olmadığında)*/
export function isTenantWebSearchReady(providers: WebSearchProviderEntity[]): boolean {
  return providers.some((p) => p.is_default);
}
