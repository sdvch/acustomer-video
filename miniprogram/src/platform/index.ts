// 平台适配层：仅支持微信小程序（weapp）。
// 封装 chooseMedia / uploadFile / saveToAlbum，页面统一从这里调用。
import Taro from '@tarojs/taro'
import { chooseAssets, uploadAsset, saveToAlbum, PickedAsset } from './wechat'

export type { PickedAsset }

export { chooseAssets, uploadAsset, saveToAlbum }

export function getPlatformName(): string {
  return 'weapp'
}

export { Taro }
