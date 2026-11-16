import { create } from 'zustand'
import type { Asset } from '../types'

interface AppState {
  assets: Asset[]
  projectId: number | null
  setAssets: (assets: Asset[]) => void
  addAssets: (assets: Asset[]) => void
  removeAsset: (id: number) => void
  reorderAssets: (assets: Asset[]) => void
  setProjectId: (id: number | null) => void
  clear: () => void
}

export const useStore = create<AppState>((set) => ({
  assets: [],
  projectId: null,
  setAssets: (assets) => set({ assets }),
  addAssets: (list) => set((s) => ({ assets: [...s.assets, ...list] })),
  removeAsset: (id) => set((s) => ({ assets: s.assets.filter((a) => a.id !== id) })),
  reorderAssets: (assets) => set({ assets }),
  setProjectId: (projectId) => set({ projectId }),
  clear: () => set({ assets: [], projectId: null }),
}))
