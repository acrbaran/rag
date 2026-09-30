import 'vitepress'

declare module 'vitepress/dist/client/theme-default/config' {
  export interface ThemeConfig {
    /** Belge sitesinde gösterilen Rethra sürümü (depo kökündeki VERSION'dan alınır) */
    rethraVersion?: string
  }
}
