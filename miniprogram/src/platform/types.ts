// 平台适配层共享类型：weapp 与 h5 两个实现共用同一份接口契约
export interface PickedAsset {
  /** weapp: 临时文件路径；h5: Object URL（配合 file 字段使用） */
  path: string
  size: number
  type: 'image' | 'video'
  mime: string
  durationMs?: number
  width?: number
  height?: number
  /** 仅 h5：原生 File 对象，上传时直接使用，避免再读一遍 Object URL */
  file?: File
}

export interface UploadAssetParams {
  filePath: string
  uploadUrl?: string
  objectKey: string
  assetId: number
  mime: string
  token: string
  apiBase: string
  localMode: boolean
  /** 仅 h5：从文件选择器拿到的原生 File */
  file?: File
}
