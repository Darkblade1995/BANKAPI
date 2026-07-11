import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '../store/authStore'
import Navbar from '../components/Navbar'
import api from '../api/axios'

interface Account {
  id: number
  userId: number
  balance: number
  currency: string
  createdAt: string
}

interface Transaction {
  id: number
  fromAccount: number
  toAccount: number
  amount: number
  fromCurrency: string
  toCurrency: string
  exchangeRate: number
  convertedAmount: number
  createdAt: string
}

export default function Dashboard() {
  const navigate = useNavigate()
  const { user } = useAuthStore()
  const [accounts, setAccounts] = useState<Account[]>([])
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [notification, setNotification] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchData()
    connectWebSocket()
  }, [])

  const fetchData = async () => {
    try {
      const accountsRes = await api.get('/accounts')
      setAccounts(accountsRes.data.accounts)

      if (accountsRes.data.accounts.length > 0) {
        const firstAccount = accountsRes.data.accounts[0]
        const txRes = await api.get(`/accounts/${firstAccount.id}/transactions`)
        setTransactions(txRes.data.transactions)
      }
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  const connectWebSocket = () => {
    const token = localStorage.getItem('accessToken')
    if (!token) return

    const ws = new WebSocket(`ws://localhost:8080/v1/ws?token=${token}`)

    ws.onmessage = (event) => {
      const msg = JSON.parse(event.data)

      if (msg.type === 'deposit') {
        setNotification(`💰 Depósito recibido: ${msg.payload.amount} ${msg.payload.currency}`)
        fetchData()
      } else if (msg.type === 'transfer_received') {
        setNotification(`📨 Transferencia recibida: ${msg.payload.amount} ${msg.payload.currency}`)
        fetchData()
      } else if (msg.type === 'transfer_sent') {
        setNotification(`📤 Transferencia enviada: ${msg.payload.amount} ${msg.payload.currency}`)
        fetchData()
      }

      setTimeout(() => setNotification(null), 5000)
    }

    ws.onerror = () => console.error('WebSocket error')
  }

  const formatBalance = (amount: number, currency: string) => {
    return new Intl.NumberFormat('es-CO', {
      style: 'currency',
      currency: currency,
      minimumFractionDigits: 0,
    }).format(amount / 100)
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-blue-600 font-medium">Cargando...</div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <Navbar />

      <div className="max-w-6xl mx-auto px-6 py-8">

        {/* Notificación WebSocket */}
        {notification && (
          <div className="bg-green-50 border border-green-200 text-green-700 rounded-xl p-4 mb-6 flex items-center gap-3 animate-pulse">
            <span className="text-lg">🔔</span>
            <span className="font-medium">{notification}</span>
          </div>
        )}

        {/* Bienvenida */}
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-gray-800">
            Hola, {user?.firstName} 👋
          </h1>
          <p className="text-gray-500 mt-1">
            Aquí está el resumen de tus cuentas
          </p>
        </div>

        {/* Cuentas */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
          {accounts.length === 0 ? (
            <div className="col-span-3 bg-white rounded-2xl p-8 text-center shadow-sm">
              <p className="text-gray-500 mb-4">No tienes cuentas aún</p>
              <button
                onClick={() => navigate('/accounts')}
                className="bg-blue-600 text-white px-6 py-2 rounded-lg hover:bg-blue-700 transition"
              >
                Crear cuenta
              </button>
            </div>
          ) : (
            accounts.map((account) => (
              <div
                key={account.id}
                onClick={() => navigate(`/accounts/${account.id}`)}
                className="bg-gradient-to-br from-blue-600 to-blue-800 rounded-2xl p-6 text-white cursor-pointer hover:shadow-lg transition"
              >
                <div className="flex justify-between items-start mb-4">
                  <span className="text-blue-200 text-sm">Cuenta #{account.id}</span>
                  <span className="bg-white/20 px-3 py-1 rounded-full text-xs font-medium">
                    {account.currency}
                  </span>
                </div>
                <p className="text-3xl font-bold mb-1">
                  {formatBalance(account.balance, account.currency)}
                </p>
                <p className="text-blue-200 text-sm">Saldo disponible</p>
              </div>
            ))
          )}
        </div>

        {/* Acciones rápidas */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
          {[
            { icon: '💸', label: 'Transferir', path: '/transfer' },
            { icon: '💰', label: 'Depositar', path: '/accounts' },
            { icon: '📊', label: 'Cuentas', path: '/accounts' },
            { icon: '📋', label: 'Historial', path: '/accounts' },
          ].map((action) => (
            <button
              key={action.label}
              onClick={() => navigate(action.path)}
              className="bg-white rounded-2xl p-4 text-center shadow-sm hover:shadow-md transition"
            >
              <span className="text-2xl block mb-2">{action.icon}</span>
              <span className="text-sm font-medium text-gray-700">{action.label}</span>
            </button>
          ))}
        </div>

        {/* Últimas transacciones */}
        <div className="bg-white rounded-2xl shadow-sm p-6">
          <h2 className="text-lg font-bold text-gray-800 mb-4">
            Últimas transacciones
          </h2>
          {transactions.length === 0 ? (
            <p className="text-gray-400 text-center py-8">No hay transacciones aún</p>
          ) : (
            <div className="space-y-3">
              {transactions.slice(0, 5).map((tx) => (
                <div
                  key={tx.id}
                  className="flex items-center justify-between p-3 rounded-xl hover:bg-gray-50 transition"
                >
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 bg-blue-100 rounded-full flex items-center justify-center">
                      <span className="text-blue-600">💳</span>
                    </div>
                    <div>
                      <p className="text-sm font-medium text-gray-800">
                        Cuenta #{tx.fromAccount} → #{tx.toAccount}
                      </p>
                      <p className="text-xs text-gray-400">
                        {new Date(tx.createdAt).toLocaleDateString('es-CO')}
                      </p>
                    </div>
                  </div>
                  <div className="text-right">
                    <p className="text-sm font-bold text-gray-800">
                      {tx.amount} {tx.fromCurrency}
                    </p>
                    {tx.fromCurrency !== tx.toCurrency && (
                      <p className="text-xs text-gray-400">
                        = {tx.convertedAmount} {tx.toCurrency}
                      </p>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

      </div>
    </div>
  )
}