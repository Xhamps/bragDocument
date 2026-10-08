function required(name: keyof ImportMetaEnv): string {
  const v = import.meta.env[name];
  if (!v) throw new Error(`Missing environment variable ${name}`);
  return v;
}

export const env = {
  apiUrl: required("VITE_API_URL"),
  supabaseUrl: required("VITE_SUPABASE_URL"),
  supabaseAnonKey: required("VITE_SUPABASE_ANON_KEY"),
};
