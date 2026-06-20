import { Provider as UrqlProvider, } from "urql";
import { graphqlClient, } from "@/lib/graphql-client";

export function GraphQLProvider({ children, }: { children: React.ReactNode },) {
  return <UrqlProvider value={graphqlClient}>{children}</UrqlProvider>;
}
