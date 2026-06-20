export async function api(path, options = {}) {
  const res = await fetch(path, options)
  if (!res.ok) {
    let msg = '请求失败'
    try { msg = (await res.json()).error || msg } catch {}
    throw new Error(msg)
  }
  return res
}
export const voices = [
  ['冰糖','活泼少女'], ['茉莉','知性女声'], ['苏打','阳光少年'], ['白桦','成熟男声'],
  ['Mia','Lively girl'], ['Chloe','Sweet Dreamy'], ['Milo','Sunny boy'], ['Dean','Steady Gentle']
]
