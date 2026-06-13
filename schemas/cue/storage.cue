package clarityit

#StorageClass: "postgres" | "nats" | "s3" | "redis" | "git" | "vector"

#StorageBoundary: {
  class: #StorageClass
  owns: [...string]
  forbidden: [...string]
}

boundaries: [...#StorageBoundary]
