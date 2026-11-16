// 平台适配层：微信小程序（weapp）与浏览器（h5）双端支持。
// 封装 chooseAssets / uploadAsset / saveToAlbum / copyText，页面统一从这里调用，
// 构建时由 TARO_ENV 决定实际实现（两个实现共享 types.ts 的接口契约）。
import Taro from '@tarojs/taro'
import * as wechat from './wechat'
import * as h5 from './h5'
import type { PickedAsset, UploadAssetParams } from './types'

export type { PickedAsset, UploadAssetParams }

const impl = process.env.TARO_ENV === 'h5' ? h5 : wechat

export const chooseAssets: (count?: number) => Promise<PickedAsset[]> = impl.chooseAssets
export const uploadAsset: (params: UploadAssetParams) => Promise<void> = impl.uploadAsset
export const saveToAlbum: (url: string) => Promise<void> = impl.saveToAlbum
export const copyText: (text: string) => Promise<void> = impl.copyText

export function getPlatformName(): 'weapp' | 'h5' {
  return process.env.TARO_ENV === 'h5' ? 'h5' : 'weapp'
}

export { Taro }
