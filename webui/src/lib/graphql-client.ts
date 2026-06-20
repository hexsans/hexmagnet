import { Client, cacheExchange, fetchExchange, mapExchange, } from "@urql/core";

const url = import.meta.env.VITE_GRAPHQL_URL || "/graphql";

const errorExchange = mapExchange({
  onError(error,) {
    console.error("[GraphQL]", error.message,);
  },
},);

export const graphqlClient = new Client({
  url,
  exchanges: [cacheExchange, errorExchange, fetchExchange,],
  preferGetMethod: false,
},);
