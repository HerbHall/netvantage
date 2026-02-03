import { useParams } from 'react-router-dom'

export function DeviceDetailPage() {
  const { id } = useParams<{ id: string }>()

  return (
    <div>
      <h1 className="text-2xl font-semibold">Device Detail</h1>
      <p className="mt-2 text-muted-foreground">Details for device {id} will appear here.</p>
    </div>
  )
}
