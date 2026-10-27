import Icon from './Icon.jsx'

const messages = {
  none: ['Your next chapter', 'Become a verified driver and share the road with others.'],
  pending: ['Verification in progress', 'Your request is under review. You can keep booking as a client while you wait.'],
  approved: ['You’re a verified driver', 'You’re ready to publish drives. Switch to Driver in your profile to get started.'],
  rejected: ['A few details need attention', 'Please check your application and submit it again.'],
}

export default function VerificationStatus({ verification }) {
  const [title, message] = messages[verification.status]
  const label = verification.status === 'none' ? 'Not verified' : verification.status.charAt(0).toUpperCase() + verification.status.slice(1)
  return <div className={`notice verification-status ${verification.status}`} role="status"><Icon name={verification.status === 'approved' ? 'shield' : verification.status === 'pending' ? 'clock' : 'info'} size={20} /><div><span className={`badge ${verification.status}`}>{label}</span><strong>{title}</strong><p>{verification.reason || message}</p></div></div>
}
