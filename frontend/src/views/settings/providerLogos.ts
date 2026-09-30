// Ayar kartının sol tarafındaki logo arama tablosu.
//
// Kaynaklar iki kategoriye ayrılır:
//   color/ —— Sağlayıcının resmi çok renkli SVG'si; doğrudan <img> ile oluşturulur ve markanın özgün renkleri korunur
//   mono/ —— Tek renkli SVG (çoğunlukla simple-icons / sağlayıcı markasından alınır); mask-image ile şu renge boyanır:
//             Kartın kendi marka rengi; böylece .store-card--<id> gibi kurallarda tanımlanan
//             düşük doygunluklu marka tonları kullanılır.
//
// Çağıran taraf (category, id) ile { mode, url } alır; bulunamadığında undefined döner,
// kart mevcut ilk harf monogramına geri döner.

const colorModules = import.meta.glob('@/assets/img/providers/color/*/*.svg', {
  eager: true,
  query: '?url',
  import: 'default',
}) as Record<string, string>;

const monoModules = import.meta.glob('@/assets/img/providers/mono/*/*.svg', {
  eager: true,
  query: '?url',
  import: 'default',
}) as Record<string, string>;

export type ProviderCategory = 'vectorstore' | 'storage' | 'websearch' | 'parser' | 'sandbox';

export type LogoMatch = {
  mode: 'color' | 'mono';
  url: string;
};

const buildLookup = (modules: Record<string, string>, segment: string) => {
  const map: Partial<Record<ProviderCategory, Record<string, string>>> = {};
  const re = new RegExp(`providers/${segment}/([^/]+)/([^/]+)\\.svg$`);
  for (const [path, url] of Object.entries(modules)) {
    const match = path.match(re);
    if (!match) continue;
    const [, category, id] = match;
    const bucket = (map[category as ProviderCategory] ||= {});
    bucket[id.toLowerCase()] = url;
  }
  return map;
};

const colorLookup = buildLookup(colorModules, 'color');
const monoLookup = buildLookup(monoModules, 'mono');

export function providerLogo(
  category: ProviderCategory,
  id: string | undefined | null,
): LogoMatch | undefined {
  if (!id) return undefined;
  const key = id.toLowerCase();
  const colorUrl = colorLookup[category]?.[key];
  if (colorUrl) return { mode: 'color', url: colorUrl };
  const monoUrl = monoLookup[category]?.[key];
  if (monoUrl) return { mode: 'mono', url: monoUrl };
  return undefined;
}
