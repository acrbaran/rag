// Klasör sürüklenirken Chrome/Edge, "sahte" dizin girdileriyle (size 0, uzantı yok) doldurur
// dataTransfer.files, klasör içindeki gerçek dosyalar yerine. Yalnızca webkitGetAsEntry()
// sürüklenen dizini özyinelemeli olarak gezebilir; bu nedenle önce onu denemek, yalnızca giriş API'si kullanılamadığında geri dönmek gerekir
// dataTransfer.files öğesine. Firefox'ta klasör sürüklenirken dataTransfer.files boştur,
// aynı şekilde entry gezme yolunun kullanılması gerekir.

const isHiddenSegment = (segment: string): boolean => segment.startsWith('.')

export const setRelativePath = (file: File, relativePath: string): void => {
  try {
    Object.defineProperty(file, 'webkitRelativePath', {
      value: relativePath,
      writable: false,
      enumerable: true,
      configurable: true,
    })
  } catch {
    // Eski Safari sürümleri File üzerinde defineProperty işlemini reddedebilir; bu dosyalar
    // göreli yol taşımaz ve düz biçimde yüklenir.
  }
}

const readAllDirEntries = (reader: { readEntries: Function }): Promise<any[]> => {
  return new Promise((resolve) => {
    const collected: any[] = []
    const readBatch = () => {
      reader.readEntries((entries: any[]) => {
        if (!entries || entries.length === 0) {
          resolve(collected)
        } else {
          collected.push(...entries)
          readBatch()
        }
      }, () => resolve(collected))
    }
    readBatch()
  })
}

export const traverseEntry = (entry: any, path: string): Promise<File[]> => {
  return new Promise((resolve) => {
    try {
      if (!entry) {
        resolve([])
        return
      }
      if (entry.isFile) {
        if (typeof entry.file !== 'function') {
          resolve([])
          return
        }
        entry.file((file: File) => {
          // Yalnızca dizin içindeki dosyalar için webkitRelativePath ayarla; bu,
          // <input webkitdirectory> davranışıyla tutarlıdır. Üst düzeyde sürüklenen dosyalar
          // normal dosya olarak yüklenmek üzere boş değeri korur.
          if (path) {
            const relativePath = `${path}/${file.name}`
            if (relativePath.split('/').some(isHiddenSegment)) {
              resolve([])
              return
            }
            setRelativePath(file, relativePath)
          }
          resolve([file])
        }, () => resolve([]))
      } else if (entry.isDirectory) {
        const dirPath = path ? `${path}/${entry.name}` : entry.name
        // Gizli dizinleri atla (.git, .DS_Store vb.)
        if (dirPath.split('/').some(isHiddenSegment)) {
          resolve([])
          return
        }
        if (typeof entry.createReader !== 'function') {
          resolve([])
          return
        }
        readAllDirEntries(entry.createReader())
          .then(children => Promise.all(
            children.map(c => traverseEntry(c, dirPath).catch(() => [] as File[])),
          ))
          .then(results => resolve(results.flat()))
          .catch(() => resolve([]))
      } else {
        resolve([])
      }
    } catch {
      resolve([])
    }
  })
}

export const collectDroppedFiles = async (event: DragEvent): Promise<File[]> => {
  const dataTransfer = event.dataTransfer
  const items = dataTransfer?.items ? Array.from(dataTransfer.items) : []
  // DataTransfer yalnızca drop eşzamanlı aşamasında kullanılabilir; geri dönüş listesi önce kopyalanmalıdır.
  const fallbackFiles = dataTransfer?.files ? Array.from(dataTransfer.files) : []

  if (items.length === 0) {
    return fallbackFiles
  }

  const fileItems = items.filter(item => item.kind === 'file')
  if (fileItems.length === 0) {
    return fallbackFiles
  }

  const pairs = fileItems.map(item => {
    try {
      return { item, entry: (item as any).webkitGetAsEntry?.() ?? null }
    } catch {
      return { item, entry: null }
    }
  })
  const usable = pairs.filter(p => p.entry != null)
  if (usable.length === 0) {
    // Tarayıcı webkitGetAsEntry desteklemiyorsa, önceden anlık görüntüsü alınmış FileList'e geri dön.
    return fallbackFiles
  }

  const results = await Promise.all(usable.map(async ({ item, entry }) => {
    try {
      if (entry.isDirectory) {
        return await traverseEntry(entry, '')
      }
      // Üst düzey dosyalar için getAsFile kullan: eşzamanlıdır ve boş webkitRelativePath değerini korur.
      const file = item.getAsFile()
      if (file) return [file]
      return await traverseEntry(entry, '')
    } catch {
      if (entry?.isDirectory) return []
      const file = item.getAsFile()
      return file ? [file] : []
    }
  }))

  // FileSystemEntry alındıysa, boş dizi dahil olmak üzere tarama sonucuna güvenin.
  // Klasör boşsa / tüm dosyalar gizliyse yeniden dataTransfer.files geri dönüşüne başvurulduğunda,
  // Chrome/Edge yeniden size 0 olan hayalet dizin girdilerini verir.
  return results.flat()
}
