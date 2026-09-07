import { InviteAccept } from "@/components/invite-accept";

export default async function InvitePage({ searchParams }: { searchParams: Promise<{ token?: string }> }) {
  const params = await searchParams;
  return <div className="login-page"><InviteAccept token={params.token ?? ""} /></div>;
}
