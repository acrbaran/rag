import gitlabIcon from '@/assets/img/datasource-gitlab.png'
import notionIcon from '@/assets/img/datasource-notion.ico'
import rssIcon from '@/assets/img/datasource-rss.svg'
import confluenceIcon from '@/assets/img/datasource-confluence.svg'

export const datasourceIconMap: Record<string, string> = {
  notion: notionIcon,
  rss: rssIcon,
  confluence: confluenceIcon,
  gitlab: gitlabIcon,
}

export function getDatasourceIconUrl(type: string): string | undefined {
  return datasourceIconMap[type]
}
