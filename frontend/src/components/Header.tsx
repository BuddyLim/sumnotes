import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'

export type User = {
  ID: string
  Name: string
  Email: string
  AvatarURL: string
  CreatedAt: string
}

export const fetchMe = async (): Promise<User> => {
  const response = await fetch('http://localhost:9999/api/me', {
    credentials: 'include',
  })

  if (!response.ok) {
    throw new Error('Failed to fetch user')
  }

  return response.json()
}

export default function Header() {
  const { data, error, isLoading } = useQuery({
    queryKey: ['me'],
    queryFn: fetchMe,
  })

  console.log('test')

  return (
    <header className="p-2 flex gap-2 bg-white text-black justify-between">
      <nav className="flex flex-row">
        <div className="px-2 font-bold">
          <Link to="/">Home</Link>
        </div>

        <div className="px-2 font-bold">
          <Link to="/demo/tanstack-query">TanStack Query</Link>
        </div>
      </nav>

      <div className="flex items-center gap-2">
        {isLoading && <div>Loading...</div>}
        {error && <div>Error</div>}
        {data && (
          <>
            <img
              src={data.AvatarURL}
              alt={data.Name}
              className="w-8 h-8 rounded-full"
              referrerPolicy="no-referrer"
            />
          </>
        )}
      </div>
    </header>
  )
}
