/**
 * API'nin dondurdugu bilgi tabani kapsamini, on yuz formunun guvenle kullanabilecegi bir diziye normalize et.
 *
 * @param ids API Key'in bilgi tabani ID'leri; tam yetkili Key icin sunucu null donebilir.
 * @returns Yeni bir bilgi tabani ID dizisi; null veya undefined, tum bilgi tabanlarini ifade eden bos bir dizi dondurur.
 */
export function normalizeAPIKeyKnowledgeBaseIDs(
  ids: readonly string[] | null | undefined,
): string[] {
  return ids ? [...ids] : []
}
