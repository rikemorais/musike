'use client'

import { useState, useRef } from 'react'
import { Upload, FileText, AlertCircle, CheckCircle2, Loader2 } from 'lucide-react'
import { api } from '@/lib/api'

interface ImportProgress {
  status: 'idle' | 'uploading' | 'processing' | 'completed' | 'error'
  message: string
  progress: number
}

export default function SpotifyDataImport() {
  const [dragOver, setDragOver] = useState(false)
  const [selectedFiles, setSelectedFiles] = useState<FileList | null>(null)
  const [importProgress, setImportProgress] = useState<ImportProgress>({
    status: 'idle',
    message: '',
    progress: 0
  })
  const fileInputRef = useRef<HTMLInputElement>(null)

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault()
    setDragOver(true)
  }

  const handleDragLeave = (e: React.DragEvent) => {
    e.preventDefault()
    setDragOver(false)
  }

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    setDragOver(false)
    const files = e.dataTransfer.files
    handleFiles(files)
  }

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files) {
      handleFiles(e.target.files)
    }
  }

  const handleFiles = (files: FileList) => {
    const validFiles = Array.from(files).filter(file => 
      file.type === 'application/zip' || 
      file.type === 'application/json' ||
      file.name.endsWith('.zip') || 
      file.name.endsWith('.json')
    )

    if (validFiles.length === 0) {
      setImportProgress({
        status: 'error',
        message: 'Por favor, selecione apenas arquivos ZIP ou JSON do Spotify',
        progress: 0
      })
      return
    }

    setSelectedFiles(files)
    setImportProgress({
      status: 'idle',
      message: `${validFiles.length} arquivo(s) selecionado(s)`,
      progress: 0
    })
  }

  const uploadFiles = async () => {
    if (!selectedFiles || selectedFiles.length === 0) return

    setImportProgress({
      status: 'uploading',
      message: 'Fazendo upload dos arquivos...',
      progress: 10
    })

    try {
      const formData = new FormData()
      Array.from(selectedFiles).forEach(file => {
        formData.append('files', file)
      })

      const response = await fetch('http://127.0.0.1:3000/api/v1/import/spotify', {
        method: 'POST',
        body: formData,
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('musike_token')}`
        }
      })

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        throw new Error(errorData.error || `Erro HTTP: ${response.status}`)
      }

      setImportProgress({
        status: 'processing',
        message: 'Processando dados do Spotify...',
        progress: 50
      })

      const result = await response.json()

      const interval = setInterval(() => {
        setImportProgress(prev => {
          if (prev.progress >= 90) {
            clearInterval(interval)
            return {
              status: 'completed',
              message: `Importação concluída! ${result.processed_tracks || 0} faixas processadas`,
              progress: 100
            }
          }
          return {
            ...prev,
            progress: prev.progress + 10
          }
        })
      }, 1000)

    } catch (error) {
      console.error('Erro no upload:', error)
      setImportProgress({
        status: 'error',
        message: 'Erro ao processar arquivos. Tente novamente.',
        progress: 0
      })
    }
  }

  const resetImport = () => {
    setSelectedFiles(null)
    setImportProgress({
      status: 'idle',
      message: '',
      progress: 0
    })
    if (fileInputRef.current) {
      fileInputRef.current.value = ''
    }
  }

  return (
    <div className="bg-gray-800 rounded-lg p-6">
      <div className="flex items-center space-x-3 mb-6">
        <Upload className="h-6 w-6 text-spotify-green" />
        <h3 className="text-xl font-semibold">Importar Dados Históricos do Spotify</h3>
      </div>

      <div className="bg-blue-900/30 border border-blue-600/50 rounded-lg p-4 mb-6">
        <h4 className="font-medium mb-2">Como obter seus dados do Spotify:</h4>
        <ol className="text-sm text-gray-300 space-y-1 ml-4 list-decimal">
          <li>Acesse <a href="https://www.spotify.com/account/privacy/" target="_blank" className="text-spotify-green underline">Privacy Settings do Spotify</a></li>
          <li>Selecione "Extended streaming history"</li>
          <li>Aguarde ~30 dias para receber o arquivo ZIP</li>
          <li>Faça upload do arquivo aqui para análise completa</li>
        </ol>
        <div className="flex items-center mt-3 p-2 bg-yellow-900/30 rounded">
          <AlertCircle className="h-4 w-4 text-yellow-500 mr-2" />
          <span className="text-xs text-yellow-200">Nunca compartilhe esses arquivos com terceiros!</span>
        </div>
      </div>

      <div
        className={`border-2 border-dashed rounded-lg p-8 text-center transition-colors ${
          dragOver 
            ? 'border-spotify-green bg-spotify-green/10' 
            : 'border-gray-600 hover:border-gray-500'
        }`}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
      >
        {importProgress.status === 'idle' && !selectedFiles && (
          <>
            <Upload className="h-12 w-12 text-gray-400 mx-auto mb-4" />
            <p className="text-lg font-medium mb-2">Arraste seus arquivos aqui</p>
            <p className="text-gray-400 mb-4">ou</p>
            <button
              onClick={() => fileInputRef.current?.click()}
              className="spotify-gradient text-white px-6 py-2 rounded-lg hover:scale-105 transition-transform"
            >
              Selecionar Arquivos
            </button>
            <p className="text-xs text-gray-400 mt-4">
              Formatos aceitos: ZIP, JSON (máx. 100MB por arquivo)
            </p>
          </>
        )}

        {selectedFiles && importProgress.status === 'idle' && (
          <div className="space-y-4">
            <FileText className="h-12 w-12 text-spotify-green mx-auto" />
            <div>
              <p className="font-medium">{selectedFiles.length} arquivo(s) selecionado(s)</p>
              <div className="text-sm text-gray-400 mt-2">
                {Array.from(selectedFiles).map(file => (
                  <div key={file.name} className="flex justify-between">
                    <span>{file.name}</span>
                    <span>{(file.size / 1024 / 1024).toFixed(1)} MB</span>
                  </div>
                ))}
              </div>
            </div>
            <div className="flex space-x-4 justify-center">
              <button
                onClick={uploadFiles}
                className="spotify-gradient text-white px-6 py-2 rounded-lg hover:scale-105 transition-transform"
              >
                Iniciar Importação
              </button>
              <button
                onClick={resetImport}
                className="bg-gray-600 text-white px-6 py-2 rounded-lg hover:bg-gray-500 transition-colors"
              >
                Cancelar
              </button>
            </div>
          </div>
        )}

        {(importProgress.status === 'uploading' || importProgress.status === 'processing') && (
          <div className="space-y-4">
            <Loader2 className="h-12 w-12 text-spotify-green mx-auto animate-spin" />
            <div>
              <p className="font-medium">{importProgress.message}</p>
              <div className="w-full bg-gray-700 rounded-full h-2 mt-4">
                <div 
                  className="bg-spotify-green h-2 rounded-full transition-all duration-300"
                  style={{ width: `${importProgress.progress}%` }}
                ></div>
              </div>
              <p className="text-sm text-gray-400 mt-2">{importProgress.progress}%</p>
            </div>
          </div>
        )}

        {importProgress.status === 'completed' && (
          <div className="space-y-4">
            <CheckCircle2 className="h-12 w-12 text-green-500 mx-auto" />
            <div>
              <p className="font-medium text-green-400">{importProgress.message}</p>
              <button
                onClick={resetImport}
                className="mt-4 spotify-gradient text-white px-6 py-2 rounded-lg hover:scale-105 transition-transform"
              >
                Importar Mais Dados
              </button>
            </div>
          </div>
        )}

        {importProgress.status === 'error' && (
          <div className="space-y-4">
            <AlertCircle className="h-12 w-12 text-red-500 mx-auto" />
            <div>
              <p className="font-medium text-red-400">{importProgress.message}</p>
              <button
                onClick={resetImport}
                className="mt-4 bg-red-600 text-white px-6 py-2 rounded-lg hover:bg-red-500 transition-colors"
              >
                Tentar Novamente
              </button>
            </div>
          </div>
        )}
      </div>

      <input
        ref={fileInputRef}
        type="file"
        multiple
        accept=".zip,.json"
        onChange={handleFileSelect}
        className="hidden"
      />
    </div>
  )
}