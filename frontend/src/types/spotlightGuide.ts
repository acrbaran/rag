export type GuidePlacement = 'right' | 'left' | 'bottom' | 'top'

export interface SpotlightGuideStep {
  key: string
  /** Vurgulama hedefinin CSS seçicisi; varsayılan olarak ortadaki kartı ifade eder*/
  target?: string
  placement?: GuidePlacement
  before?: () => void | Promise<void>
  /** Hedef mevcut değilse bu adımın atlanıp atlanmayacağı*/
  optional?: boolean
  /** true olduğunda kullanıcıyı vurgulanan alana doğrudan tıklamaya yönlendirir, "Sonraki" gösterilmez*/
  interact?: boolean
}
