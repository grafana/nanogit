# 🚀 Performance Benchmark Report

**Generated:** 2025-06-19T14:05:00+02:00  
**Total Benchmarks:** 168

## 📊 Performance Overview

| Operation | Speed Winner | Duration | In-Memory Winner | Memory Usage |
|-----------|--------------|----------|------------------|-------------|
| BulkCreateFiles_bulk_1000_files_medium | 🚀 nanogit | 1.52s | 💚 nanogit | 4.5 MB |
| BulkCreateFiles_bulk_1000_files_small | 🚀 nanogit | 1.49s | 💚 nanogit | 1.5 MB |
| BulkCreateFiles_bulk_100_files_medium | 🚀 nanogit | 125.8ms | 💚 nanogit | 2.9 MB |
| BulkCreateFiles_bulk_100_files_small | 🚀 nanogit | 124.2ms | 💚 nanogit | 971.5 KB |
| CompareCommits_adjacent_commits_large | 🚀 nanogit | 115.3ms | 💚 nanogit | 3.3 MB |
| CompareCommits_adjacent_commits_medium | 🚀 nanogit | 110.7ms | 💚 nanogit | 953.9 KB |
| CompareCommits_adjacent_commits_small | 🐹 go-git | 80.1ms | 💚 nanogit | 553.6 KB |
| CompareCommits_adjacent_commits_xlarge | 🚀 nanogit | 183.2ms | 💚 nanogit | 15.4 MB |
| CompareCommits_few_commits_large | 🚀 nanogit | 199.7ms | 💚 nanogit | 4.2 MB |
| CompareCommits_few_commits_medium | 🚀 nanogit | 165.7ms | 💚 nanogit | 2.3 MB |
| CompareCommits_few_commits_small | 🐹 go-git | 68.5ms | 💚 nanogit | 258.9 KB |
| CompareCommits_few_commits_xlarge | 🚀 nanogit | 360.0ms | 💚 nanogit | 17.5 MB |
| CompareCommits_max_commits_large | 🚀 nanogit | 316.1ms | 💚 nanogit | 4.1 MB |
| CompareCommits_max_commits_medium | 🚀 nanogit | 270.2ms | 💚 nanogit | 1.8 MB |
| CompareCommits_max_commits_small | 🐹 go-git | 85.3ms | 💚 nanogit | 1.3 MB |
| CompareCommits_max_commits_xlarge | 🚀 nanogit | 511.4ms | 💚 nanogit | 14.3 MB |
| CreateFile_large_repo | 🚀 nanogit | 60.7ms | 💚 nanogit | 5.0 MB |
| CreateFile_medium_repo | 🚀 nanogit | 52.6ms | 💚 nanogit | 3.3 MB |
| CreateFile_small_repo | 🚀 nanogit | 55.7ms | 💚 nanogit | 1.1 MB |
| CreateFile_xlarge_repo | 🚀 nanogit | 88.6ms | 💚 nanogit | 12.5 MB |
| DeleteFile_large_repo | 🚀 nanogit | 57.2ms | 💚 nanogit | 3.4 MB |
| DeleteFile_medium_repo | 🚀 nanogit | 53.7ms | 💚 nanogit | 2.3 MB |
| DeleteFile_small_repo | 🚀 nanogit | 48.0ms | 💚 nanogit | 2.2 MB |
| DeleteFile_xlarge_repo | 🚀 nanogit | 91.1ms | 💚 nanogit | 12.1 MB |
| GetFlatTree_large_tree | 🚀 nanogit | 56.9ms | 💚 nanogit | 2.3 MB |
| GetFlatTree_medium_tree | 🚀 nanogit | 50.4ms | 💚 nanogit | 611.4 KB |
| GetFlatTree_small_tree | 🚀 nanogit | 46.7ms | 💚 nanogit | 366.7 KB |
| GetFlatTree_xlarge_tree | 🚀 nanogit | 81.8ms | 💚 nanogit | 10.2 MB |
| UpdateFile_large_repo | 🚀 nanogit | 56.7ms | 💚 nanogit | 4.2 MB |
| UpdateFile_medium_repo | 🚀 nanogit | 52.7ms | 💚 nanogit | 2.6 MB |
| UpdateFile_small_repo | 🚀 nanogit | 47.3ms | 💚 nanogit | 1.4 MB |
| UpdateFile_xlarge_repo | 🚀 nanogit | 86.4ms | 💚 nanogit | 10.8 MB |

## ⚡ Duration Comparison

| Operation | git-cli | go-git | nanogit |
|-----------|-----------|-----------|-----------|
| BulkCreateFiles_bulk_1000_files_medium | 14.03s 🐌 | 72.33s 🐌 | 1.52s 🏆 |
| BulkCreateFiles_bulk_1000_files_small | 14.17s 🐌 | 19.12s 🐌 | 1.49s 🏆 |
| BulkCreateFiles_bulk_100_files_medium | 2.43s 🐌 | 6.59s 🐌 | 125.8ms 🏆 |
| BulkCreateFiles_bulk_100_files_small | 1.69s 🐌 | 907.4ms 🐌 | 124.2ms 🏆 |
| CompareCommits_adjacent_commits_large | 1.62s 🐌 | 2.64s 🐌 | 115.3ms 🏆 |
| CompareCommits_adjacent_commits_medium | 861.3ms 🐌 | 422.8ms | 110.7ms 🏆 |
| CompareCommits_adjacent_commits_small | 443.3ms 🐌 | 80.1ms 🏆 | 84.5ms ✅ |
| CompareCommits_adjacent_commits_xlarge | 12.90s 🐌 | 20.76s 🐌 | 183.2ms 🏆 |
| CompareCommits_few_commits_large | 3.85s 🐌 | 2.61s 🐌 | 199.7ms 🏆 |
| CompareCommits_few_commits_medium | 960.1ms 🐌 | 406.4ms | 165.7ms 🏆 |
| CompareCommits_few_commits_small | 443.6ms 🐌 | 68.5ms 🏆 | 167.7ms |
| CompareCommits_few_commits_xlarge | 7.14s 🐌 | 20.66s 🐌 | 360.0ms 🏆 |
| CompareCommits_max_commits_large | 3.89s 🐌 | 2.66s 🐌 | 316.1ms 🏆 |
| CompareCommits_max_commits_medium | 1.00s | 425.9ms ✅ | 270.2ms 🏆 |
| CompareCommits_max_commits_small | 440.0ms 🐌 | 85.3ms 🏆 | 258.1ms |
| CompareCommits_max_commits_xlarge | 10.04s 🐌 | 20.53s 🐌 | 511.4ms 🏆 |
| CreateFile_large_repo | 1.96s 🐌 | 2.96s 🐌 | 60.7ms 🏆 |
| CreateFile_medium_repo | 1.02s 🐌 | 508.0ms 🐌 | 52.6ms 🏆 |
| CreateFile_small_repo | 872.2ms 🐌 | 109.8ms ✅ | 55.7ms 🏆 |
| CreateFile_xlarge_repo | 12.77s 🐌 | 22.65s 🐌 | 88.6ms 🏆 |
| DeleteFile_large_repo | 1.90s 🐌 | 2.95s 🐌 | 57.2ms 🏆 |
| DeleteFile_medium_repo | 1.03s 🐌 | 498.9ms 🐌 | 53.7ms 🏆 |
| DeleteFile_small_repo | 853.1ms 🐌 | 105.0ms | 48.0ms 🏆 |
| DeleteFile_xlarge_repo | 10.45s 🐌 | 22.69s 🐌 | 91.1ms 🏆 |
| GetFlatTree_large_tree | 1.69s 🐌 | 2.72s 🐌 | 56.9ms 🏆 |
| GetFlatTree_medium_tree | 566.1ms 🐌 | 467.0ms 🐌 | 50.4ms 🏆 |
| GetFlatTree_small_tree | 397.0ms 🐌 | 79.7ms ✅ | 46.7ms 🏆 |
| GetFlatTree_xlarge_tree | 8.28s 🐌 | 20.72s 🐌 | 81.8ms 🏆 |
| UpdateFile_large_repo | 1.88s 🐌 | 2.95s 🐌 | 56.7ms 🏆 |
| UpdateFile_medium_repo | 1.03s 🐌 | 501.9ms 🐌 | 52.7ms 🏆 |
| UpdateFile_small_repo | 853.1ms 🐌 | 106.2ms | 47.3ms 🏆 |
| UpdateFile_xlarge_repo | 10.15s 🐌 | 22.63s 🐌 | 86.4ms 🏆 |

## 💾 Memory Usage Comparison

*Note: git-cli uses disk storage rather than keeping data in memory, so memory comparisons focus on in-memory clients (nanogit vs go-git)*

| Operation | git-cli | go-git | nanogit |
|-----------|-----------|-----------|-----------|
| BulkCreateFiles_bulk_1000_files_medium | -686864 B 💾 | 38.9 MB 🔥 | 4.5 MB 🏆 |
| BulkCreateFiles_bulk_1000_files_small | 159.9 KB 💾 | 5.4 MB | 1.5 MB 🏆 |
| BulkCreateFiles_bulk_100_files_medium | -1026344 B 💾 | 38.7 MB 🔥 | 2.9 MB 🏆 |
| BulkCreateFiles_bulk_100_files_small | -1023328 B 💾 | 5.2 MB 🔥 | 971.5 KB 🏆 |
| CompareCommits_adjacent_commits_large | 70.2 KB 💾 | 210.8 MB 🔥 | 3.3 MB 🏆 |
| CompareCommits_adjacent_commits_medium | 70.2 KB 💾 | 39.2 MB 🔥 | 953.9 KB 🏆 |
| CompareCommits_adjacent_commits_small | 70.6 KB 💾 | 3.6 MB 🔥 | 553.6 KB 🏆 |
| CompareCommits_adjacent_commits_xlarge | 70.5 KB 💾 | 1.5 GB 🔥 | 15.4 MB 🏆 |
| CompareCommits_few_commits_large | 70.2 KB 💾 | 186.5 MB 🔥 | 4.2 MB 🏆 |
| CompareCommits_few_commits_medium | 70.5 KB 💾 | 40.4 MB 🔥 | 2.3 MB 🏆 |
| CompareCommits_few_commits_small | 70.5 KB 💾 | 3.3 MB 🔥 | 258.9 KB 🏆 |
| CompareCommits_few_commits_xlarge | 70.5 KB 💾 | 1.7 GB 🔥 | 17.5 MB 🏆 |
| CompareCommits_max_commits_large | 70.2 KB 💾 | 207.5 MB 🔥 | 4.1 MB 🏆 |
| CompareCommits_max_commits_medium | 70.5 KB 💾 | 38.7 MB 🔥 | 1.8 MB 🏆 |
| CompareCommits_max_commits_small | 70.5 KB 💾 | 3.3 MB | 1.3 MB 🏆 |
| CompareCommits_max_commits_xlarge | 70.2 KB 💾 | 1.7 GB 🔥 | 14.3 MB 🏆 |
| CreateFile_large_repo | 136.1 KB 💾 | 178.5 MB 🔥 | 5.0 MB 🏆 |
| CreateFile_medium_repo | 135.8 KB 💾 | 36.5 MB 🔥 | 3.3 MB 🏆 |
| CreateFile_small_repo | 136.2 KB 💾 | 4.2 MB | 1.1 MB 🏆 |
| CreateFile_xlarge_repo | 136.2 KB 💾 | 2.1 GB 🔥 | 12.5 MB 🏆 |
| DeleteFile_large_repo | 136.1 KB 💾 | 176.5 MB 🔥 | 3.4 MB 🏆 |
| DeleteFile_medium_repo | 135.8 KB 💾 | 33.3 MB 🔥 | 2.3 MB 🏆 |
| DeleteFile_small_repo | 136.6 KB 💾 | 4.1 MB ✅ | 2.2 MB 🏆 |
| DeleteFile_xlarge_repo | 136.1 KB 💾 | 2.1 GB 🔥 | 12.1 MB 🏆 |
| GetFlatTree_large_tree | 3.2 MB 💾 | 280.6 MB 🔥 | 2.3 MB 🏆 |
| GetFlatTree_medium_tree | 740.9 KB 💾 | 44.9 MB 🔥 | 611.4 KB 🏆 |
| GetFlatTree_small_tree | 155.8 KB 💾 | 4.0 MB 🔥 | 366.7 KB 🏆 |
| GetFlatTree_xlarge_tree | 18.7 MB 💾 | 1.7 GB 🔥 | 10.2 MB 🏆 |
| UpdateFile_large_repo | 135.4 KB 💾 | 254.4 MB 🔥 | 4.2 MB 🏆 |
| UpdateFile_medium_repo | 135.0 KB 💾 | 38.9 MB 🔥 | 2.6 MB 🏆 |
| UpdateFile_small_repo | 135.4 KB 💾 | 3.5 MB | 1.4 MB 🏆 |
| UpdateFile_xlarge_repo | 135.4 KB 💾 | 2.1 GB 🔥 | 10.8 MB 🏆 |

## 🎯 Nanogit Performance Analysis

### ⚡ Speed Comparison

| Operation | vs git-cli | vs go-git |
|-----------|-----------|-----------|
| BulkCreateFiles_bulk_1000_files_medium | 9.2x faster 🚀 | 47.7x faster 🚀 |
| BulkCreateFiles_bulk_1000_files_small | 9.5x faster 🚀 | 12.8x faster 🚀 |
| BulkCreateFiles_bulk_100_files_medium | 19.3x faster 🚀 | 52.4x faster 🚀 |
| BulkCreateFiles_bulk_100_files_small | 13.6x faster 🚀 | 7.3x faster 🚀 |
| CompareCommits_adjacent_commits_large | 14.1x faster 🚀 | 22.9x faster 🚀 |
| CompareCommits_adjacent_commits_medium | 7.8x faster 🚀 | 3.8x faster 🚀 |
| CompareCommits_adjacent_commits_small | 5.2x faster 🚀 | ~same ⚖️ |
| CompareCommits_adjacent_commits_xlarge | 70.4x faster 🚀 | 113.3x faster 🚀 |
| CompareCommits_few_commits_large | 19.3x faster 🚀 | 13.1x faster 🚀 |
| CompareCommits_few_commits_medium | 5.8x faster 🚀 | 2.5x faster 🚀 |
| CompareCommits_few_commits_small | 2.6x faster 🚀 | 2.4x slower 🐌 |
| CompareCommits_few_commits_xlarge | 19.8x faster 🚀 | 57.4x faster 🚀 |
| CompareCommits_max_commits_large | 12.3x faster 🚀 | 8.4x faster 🚀 |
| CompareCommits_max_commits_medium | 3.7x faster 🚀 | 1.6x faster ✅ |
| CompareCommits_max_commits_small | 1.7x faster ✅ | 3.0x slower 🐌 |
| CompareCommits_max_commits_xlarge | 19.6x faster 🚀 | 40.1x faster 🚀 |
| CreateFile_large_repo | 32.2x faster 🚀 | 48.7x faster 🚀 |
| CreateFile_medium_repo | 19.3x faster 🚀 | 9.6x faster 🚀 |
| CreateFile_small_repo | 15.7x faster 🚀 | 2.0x faster ✅ |
| CreateFile_xlarge_repo | 144.1x faster 🚀 | 255.7x faster 🚀 |
| DeleteFile_large_repo | 33.3x faster 🚀 | 51.6x faster 🚀 |
| DeleteFile_medium_repo | 19.2x faster 🚀 | 9.3x faster 🚀 |
| DeleteFile_small_repo | 17.8x faster 🚀 | 2.2x faster 🚀 |
| DeleteFile_xlarge_repo | 114.6x faster 🚀 | 249.0x faster 🚀 |
| GetFlatTree_large_tree | 29.7x faster 🚀 | 47.7x faster 🚀 |
| GetFlatTree_medium_tree | 11.2x faster 🚀 | 9.3x faster 🚀 |
| GetFlatTree_small_tree | 8.5x faster 🚀 | 1.7x faster ✅ |
| GetFlatTree_xlarge_tree | 101.2x faster 🚀 | 253.4x faster 🚀 |
| UpdateFile_large_repo | 33.1x faster 🚀 | 52.0x faster 🚀 |
| UpdateFile_medium_repo | 19.6x faster 🚀 | 9.5x faster 🚀 |
| UpdateFile_small_repo | 18.0x faster 🚀 | 2.2x faster 🚀 |
| UpdateFile_xlarge_repo | 117.4x faster 🚀 | 261.8x faster 🚀 |

### 💾 Memory Comparison

*Note: git-cli uses minimal memory as it stores data on disk, not in memory*

| Operation | vs git-cli | vs go-git |
|-----------|-----------|-----------|
| BulkCreateFiles_bulk_1000_files_medium | -6.9x more 💾 | 8.6x less 💚 |
| BulkCreateFiles_bulk_1000_files_small | 9.7x more 💾 | 3.6x less 💚 |
| BulkCreateFiles_bulk_100_files_medium | -3.0x more 💾 | 13.2x less 💚 |
| BulkCreateFiles_bulk_100_files_small | -1.0x more 💾 | 5.5x less 💚 |
| CompareCommits_adjacent_commits_large | 48.6x more 💾 | 63.2x less 💚 |
| CompareCommits_adjacent_commits_medium | 13.6x more 💾 | 42.1x less 💚 |
| CompareCommits_adjacent_commits_small | 7.8x more 💾 | 6.6x less 💚 |
| CompareCommits_adjacent_commits_xlarge | 223.5x more 💾 | 103.0x less 💚 |
| CompareCommits_few_commits_large | 60.9x more 💾 | 44.7x less 💚 |
| CompareCommits_few_commits_medium | 33.5x more 💾 | 17.5x less 💚 |
| CompareCommits_few_commits_small | 3.7x more 💾 | 12.9x less 💚 |
| CompareCommits_few_commits_xlarge | 253.4x more 💾 | 101.2x less 💚 |
| CompareCommits_max_commits_large | 59.6x more 💾 | 50.8x less 💚 |
| CompareCommits_max_commits_medium | 25.6x more 💾 | 22.0x less 💚 |
| CompareCommits_max_commits_small | 18.5x more 💾 | 2.6x less 💚 |
| CompareCommits_max_commits_xlarge | 208.2x more 💾 | 123.4x less 💚 |
| CreateFile_large_repo | 37.4x more 💾 | 35.9x less 💚 |
| CreateFile_medium_repo | 25.2x more 💾 | 10.9x less 💚 |
| CreateFile_small_repo | 8.4x more 💾 | 3.7x less 💚 |
| CreateFile_xlarge_repo | 94.3x more 💾 | 169.1x less 💚 |
| DeleteFile_large_repo | 25.6x more 💾 | 51.8x less 💚 |
| DeleteFile_medium_repo | 17.4x more 💾 | 14.4x less 💚 |
| DeleteFile_small_repo | 16.5x more 💾 | 1.9x less ✅ |
| DeleteFile_xlarge_repo | 91.3x more 💾 | 175.2x less 💚 |
| GetFlatTree_large_tree | 0.7x more 💾 | 123.2x less 💚 |
| GetFlatTree_medium_tree | 0.8x more 💾 | 75.3x less 💚 |
| GetFlatTree_small_tree | 2.4x more 💾 | 11.1x less 💚 |
| GetFlatTree_xlarge_tree | 0.5x more 💾 | 170.3x less 💚 |
| UpdateFile_large_repo | 31.6x more 💾 | 60.8x less 💚 |
| UpdateFile_medium_repo | 19.5x more 💾 | 15.2x less 💚 |
| UpdateFile_small_repo | 10.8x more 💾 | 2.5x less 💚 |
| UpdateFile_xlarge_repo | 81.5x more 💾 | 197.8x less 💚 |

## 📈 Detailed Statistics

### BulkCreateFiles_bulk_1000_files_medium

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 14.03s | 14.03s | -686864 B | -686864 B |
| go-git | 1 | ✅ 100.0% | 72.33s | 72.33s | 38.9 MB | 38.9 MB |
| nanogit | 1 | ✅ 100.0% | 1.52s | 1.52s | 4.5 MB | 4.5 MB |

### BulkCreateFiles_bulk_1000_files_small

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 14.17s | 14.17s | 159.9 KB | 159.9 KB |
| go-git | 1 | ✅ 100.0% | 19.12s | 19.12s | 5.4 MB | 5.4 MB |
| nanogit | 1 | ✅ 100.0% | 1.49s | 1.49s | 1.5 MB | 1.5 MB |

### BulkCreateFiles_bulk_100_files_medium

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 2.43s | 2.43s | -1026344 B | -1026344 B |
| go-git | 1 | ✅ 100.0% | 6.59s | 6.59s | 38.7 MB | 38.7 MB |
| nanogit | 1 | ✅ 100.0% | 125.8ms | 125.8ms | 2.9 MB | 2.9 MB |

### BulkCreateFiles_bulk_100_files_small

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 1.69s | 1.69s | -1023328 B | -1023328 B |
| go-git | 1 | ✅ 100.0% | 907.4ms | 907.4ms | 5.2 MB | 5.2 MB |
| nanogit | 1 | ✅ 100.0% | 124.2ms | 124.2ms | 971.5 KB | 971.5 KB |

### CompareCommits_adjacent_commits_large

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 1.62s | 1.62s | 70.2 KB | 70.2 KB |
| go-git | 1 | ⚠️ 0.0% | 2.64s | 2.64s | 210.8 MB | 210.8 MB |
| nanogit | 1 | ✅ 100.0% | 115.3ms | 115.3ms | 3.3 MB | 3.3 MB |

### CompareCommits_adjacent_commits_medium

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 861.3ms | 861.3ms | 70.2 KB | 70.2 KB |
| go-git | 1 | ⚠️ 0.0% | 422.8ms | 422.8ms | 39.2 MB | 39.2 MB |
| nanogit | 1 | ✅ 100.0% | 110.7ms | 110.7ms | 953.9 KB | 953.9 KB |

### CompareCommits_adjacent_commits_small

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 443.3ms | 443.3ms | 70.6 KB | 70.6 KB |
| go-git | 1 | ⚠️ 0.0% | 80.1ms | 80.1ms | 3.6 MB | 3.6 MB |
| nanogit | 1 | ✅ 100.0% | 84.5ms | 84.5ms | 553.6 KB | 553.6 KB |

### CompareCommits_adjacent_commits_xlarge

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 12.90s | 12.90s | 70.5 KB | 70.5 KB |
| go-git | 1 | ⚠️ 0.0% | 20.76s | 20.76s | 1.5 GB | 1.5 GB |
| nanogit | 1 | ✅ 100.0% | 183.2ms | 183.2ms | 15.4 MB | 15.4 MB |

### CompareCommits_few_commits_large

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 3.85s | 3.85s | 70.2 KB | 70.2 KB |
| go-git | 1 | ⚠️ 0.0% | 2.61s | 2.61s | 186.5 MB | 186.5 MB |
| nanogit | 1 | ✅ 100.0% | 199.7ms | 199.7ms | 4.2 MB | 4.2 MB |

### CompareCommits_few_commits_medium

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 960.1ms | 960.1ms | 70.5 KB | 70.5 KB |
| go-git | 1 | ⚠️ 0.0% | 406.4ms | 406.4ms | 40.4 MB | 40.4 MB |
| nanogit | 1 | ✅ 100.0% | 165.7ms | 165.7ms | 2.3 MB | 2.3 MB |

### CompareCommits_few_commits_small

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 443.6ms | 443.6ms | 70.5 KB | 70.5 KB |
| go-git | 1 | ⚠️ 0.0% | 68.5ms | 68.5ms | 3.3 MB | 3.3 MB |
| nanogit | 1 | ✅ 100.0% | 167.7ms | 167.7ms | 258.9 KB | 258.9 KB |

### CompareCommits_few_commits_xlarge

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 7.14s | 7.14s | 70.5 KB | 70.5 KB |
| go-git | 1 | ⚠️ 0.0% | 20.66s | 20.66s | 1.7 GB | 1.7 GB |
| nanogit | 1 | ✅ 100.0% | 360.0ms | 360.0ms | 17.5 MB | 17.5 MB |

### CompareCommits_max_commits_large

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 3.89s | 3.89s | 70.2 KB | 70.2 KB |
| go-git | 1 | ⚠️ 0.0% | 2.66s | 2.66s | 207.5 MB | 207.5 MB |
| nanogit | 1 | ✅ 100.0% | 316.1ms | 316.1ms | 4.1 MB | 4.1 MB |

### CompareCommits_max_commits_medium

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 1.00s | 1.00s | 70.5 KB | 70.5 KB |
| go-git | 1 | ⚠️ 0.0% | 425.9ms | 425.9ms | 38.7 MB | 38.7 MB |
| nanogit | 1 | ✅ 100.0% | 270.2ms | 270.2ms | 1.8 MB | 1.8 MB |

### CompareCommits_max_commits_small

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 440.0ms | 440.0ms | 70.5 KB | 70.5 KB |
| go-git | 1 | ⚠️ 0.0% | 85.3ms | 85.3ms | 3.3 MB | 3.3 MB |
| nanogit | 1 | ✅ 100.0% | 258.1ms | 258.1ms | 1.3 MB | 1.3 MB |

### CompareCommits_max_commits_xlarge

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 10.04s | 10.04s | 70.2 KB | 70.2 KB |
| go-git | 1 | ⚠️ 0.0% | 20.53s | 20.53s | 1.7 GB | 1.7 GB |
| nanogit | 1 | ✅ 100.0% | 511.4ms | 511.4ms | 14.3 MB | 14.3 MB |

### CreateFile_large_repo

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 3 | ✅ 100.0% | 1.96s | 2.00s | 136.1 KB | 135.9 KB |
| go-git | 3 | ✅ 100.0% | 2.96s | 2.98s | 178.5 MB | 178.1 MB |
| nanogit | 3 | ✅ 100.0% | 60.7ms | 65.7ms | 5.0 MB | 5.0 MB |

### CreateFile_medium_repo

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 3 | ✅ 100.0% | 1.02s | 1.04s | 135.8 KB | 135.9 KB |
| go-git | 3 | ✅ 100.0% | 508.0ms | 520.0ms | 36.5 MB | 32.6 MB |
| nanogit | 3 | ✅ 100.0% | 52.6ms | 55.0ms | 3.3 MB | 3.6 MB |

### CreateFile_small_repo

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 3 | ✅ 100.0% | 872.2ms | 925.6ms | 136.2 KB | 136.2 KB |
| go-git | 3 | ✅ 100.0% | 109.8ms | 129.0ms | 4.2 MB | 4.5 MB |
| nanogit | 3 | ✅ 100.0% | 55.7ms | 75.3ms | 1.1 MB | 929.4 KB |

### CreateFile_xlarge_repo

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 3 | ✅ 100.0% | 12.77s | 20.26s | 136.2 KB | 135.9 KB |
| go-git | 3 | ✅ 100.0% | 22.65s | 22.82s | 2.1 GB | 2.1 GB |
| nanogit | 3 | ✅ 100.0% | 88.6ms | 91.4ms | 12.5 MB | 12.9 MB |

### DeleteFile_large_repo

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 3 | ✅ 100.0% | 1.90s | 1.99s | 136.1 KB | 135.8 KB |
| go-git | 3 | ✅ 100.0% | 2.95s | 2.99s | 176.5 MB | 176.7 MB |
| nanogit | 3 | ✅ 100.0% | 57.2ms | 59.4ms | 3.4 MB | 3.4 MB |

### DeleteFile_medium_repo

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 3 | ✅ 100.0% | 1.03s | 1.04s | 135.8 KB | 135.8 KB |
| go-git | 3 | ✅ 100.0% | 498.9ms | 499.9ms | 33.3 MB | 29.6 MB |
| nanogit | 3 | ✅ 100.0% | 53.7ms | 56.3ms | 2.3 MB | 2.0 MB |

### DeleteFile_small_repo

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 3 | ✅ 100.0% | 853.1ms | 869.2ms | 136.6 KB | 135.8 KB |
| go-git | 3 | ✅ 100.0% | 105.0ms | 110.1ms | 4.1 MB | 4.0 MB |
| nanogit | 3 | ✅ 100.0% | 48.0ms | 51.4ms | 2.2 MB | 2.5 MB |

### DeleteFile_xlarge_repo

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 3 | ✅ 100.0% | 10.45s | 10.84s | 136.1 KB | 136.2 KB |
| go-git | 3 | ✅ 100.0% | 22.69s | 22.82s | 2.1 GB | 2.1 GB |
| nanogit | 3 | ✅ 100.0% | 91.1ms | 100.9ms | 12.1 MB | 12.1 MB |

### GetFlatTree_large_tree

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 1.69s | 1.69s | 3.2 MB | 3.2 MB |
| go-git | 1 | ✅ 100.0% | 2.72s | 2.72s | 280.6 MB | 280.6 MB |
| nanogit | 1 | ✅ 100.0% | 56.9ms | 56.9ms | 2.3 MB | 2.3 MB |

### GetFlatTree_medium_tree

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 566.1ms | 566.1ms | 740.9 KB | 740.9 KB |
| go-git | 1 | ✅ 100.0% | 467.0ms | 467.0ms | 44.9 MB | 44.9 MB |
| nanogit | 1 | ✅ 100.0% | 50.4ms | 50.4ms | 611.4 KB | 611.4 KB |

### GetFlatTree_small_tree

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 397.0ms | 397.0ms | 155.8 KB | 155.8 KB |
| go-git | 1 | ✅ 100.0% | 79.7ms | 79.7ms | 4.0 MB | 4.0 MB |
| nanogit | 1 | ✅ 100.0% | 46.7ms | 46.7ms | 366.7 KB | 366.7 KB |

### GetFlatTree_xlarge_tree

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 8.28s | 8.28s | 18.7 MB | 18.7 MB |
| go-git | 1 | ✅ 100.0% | 20.72s | 20.72s | 1.7 GB | 1.7 GB |
| nanogit | 1 | ✅ 100.0% | 81.8ms | 81.8ms | 10.2 MB | 10.2 MB |

### UpdateFile_large_repo

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 3 | ✅ 100.0% | 1.88s | 1.97s | 135.4 KB | 135.6 KB |
| go-git | 3 | ✅ 100.0% | 2.95s | 2.96s | 254.4 MB | 291.0 MB |
| nanogit | 3 | ✅ 100.0% | 56.7ms | 58.6ms | 4.2 MB | 4.2 MB |

### UpdateFile_medium_repo

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 3 | ✅ 100.0% | 1.03s | 1.04s | 135.0 KB | 135.1 KB |
| go-git | 3 | ✅ 100.0% | 501.9ms | 515.9ms | 38.9 MB | 32.3 MB |
| nanogit | 3 | ✅ 100.0% | 52.7ms | 59.3ms | 2.6 MB | 2.8 MB |

### UpdateFile_small_repo

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 3 | ✅ 100.0% | 853.1ms | 872.9ms | 135.4 KB | 135.1 KB |
| go-git | 3 | ✅ 100.0% | 106.2ms | 112.0ms | 3.5 MB | 3.6 MB |
| nanogit | 3 | ✅ 100.0% | 47.3ms | 47.8ms | 1.4 MB | 930.4 KB |

### UpdateFile_xlarge_repo

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 3 | ✅ 100.0% | 10.15s | 10.76s | 135.4 KB | 135.6 KB |
| go-git | 3 | ✅ 100.0% | 22.63s | 22.65s | 2.1 GB | 2.1 GB |
| nanogit | 3 | ✅ 100.0% | 86.4ms | 91.0ms | 10.8 MB | 9.7 MB |

