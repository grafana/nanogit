# 🚀 Performance Benchmark Report

**Generated:** 2025-07-08T19:17:28+02:00  
**Total Benchmarks:** 168

## 📊 Performance Overview

| Operation                              | Speed Winner | Duration | In-Memory Winner | Memory Usage |
| -------------------------------------- | ------------ | -------- | ---------------- | ------------ |
| BulkCreateFiles_bulk_1000_files_medium | 🚀 nanogit    | 115.7ms  | 💚 nanogit        | 3.8 MB       |
| BulkCreateFiles_bulk_1000_files_small  | 🚀 nanogit    | 111.1ms  | 💚 nanogit        | 3.1 MB       |
| BulkCreateFiles_bulk_100_files_medium  | 🚀 nanogit    | 83.0ms   | 💚 nanogit        | 2.4 MB       |
| BulkCreateFiles_bulk_100_files_small   | 🚀 nanogit    | 76.8ms   | 💚 nanogit        | 1.7 MB       |
| CompareCommits_adjacent_commits_large  | 🚀 nanogit    | 100.1ms  | 💚 nanogit        | 6.3 MB       |
| CompareCommits_adjacent_commits_medium | 🚀 nanogit    | 82.7ms   | 💚 nanogit        | 2.8 MB       |
| CompareCommits_adjacent_commits_small  | 🚀 nanogit    | 67.6ms   | 💚 nanogit        | 1.6 MB       |
| CompareCommits_adjacent_commits_xlarge | 🚀 nanogit    | 111.5ms  | 💚 nanogit        | 16.0 MB      |
| CompareCommits_few_commits_large       | 🚀 nanogit    | 154.9ms  | 💚 nanogit        | 7.3 MB       |
| CompareCommits_few_commits_medium      | 🚀 nanogit    | 157.6ms  | 💚 nanogit        | 2.3 MB       |
| CompareCommits_few_commits_small       | 🐹 go-git     | 68.6ms   | 💚 nanogit        | 2.4 MB       |
| CompareCommits_few_commits_xlarge      | 🚀 nanogit    | 209.4ms  | 💚 nanogit        | 17.5 MB      |
| CompareCommits_max_commits_large       | 🚀 nanogit    | 253.5ms  | 💚 nanogit        | 7.3 MB       |
| CompareCommits_max_commits_medium      | 🚀 nanogit    | 235.8ms  | 💚 nanogit        | 2.6 MB       |
| CompareCommits_max_commits_small       | 🐹 go-git     | 68.3ms   | 💚 nanogit        | 3.0 MB       |
| CompareCommits_max_commits_xlarge      | 🚀 nanogit    | 336.7ms  | 💚 nanogit        | 16.9 MB      |
| CreateFile_large_repo                  | 🚀 nanogit    | 63.5ms   | 💚 nanogit        | 3.5 MB       |
| CreateFile_medium_repo                 | 🚀 nanogit    | 60.6ms   | 💚 nanogit        | 1.5 MB       |
| CreateFile_small_repo                  | 🚀 nanogit    | 52.2ms   | 💚 nanogit        | 1.4 MB       |
| CreateFile_xlarge_repo                 | 🚀 nanogit    | 80.7ms   | 💚 nanogit        | 11.2 MB      |
| DeleteFile_large_repo                  | 🚀 nanogit    | 64.7ms   | 💚 nanogit        | 3.4 MB       |
| DeleteFile_medium_repo                 | 🚀 nanogit    | 55.2ms   | 💚 nanogit        | 1.5 MB       |
| DeleteFile_small_repo                  | 🚀 nanogit    | 51.9ms   | 💚 nanogit        | 1.6 MB       |
| DeleteFile_xlarge_repo                 | 🚀 nanogit    | 77.4ms   | 💚 nanogit        | 11.4 MB      |
| GetFlatTree_large_tree                 | 🚀 nanogit    | 58.4ms   | 💚 nanogit        | 3.6 MB       |
| GetFlatTree_medium_tree                | 🚀 nanogit    | 53.3ms   | 💚 nanogit        | 1.3 MB       |
| GetFlatTree_small_tree                 | 🚀 nanogit    | 52.9ms   | 💚 nanogit        | 697.8 KB     |
| GetFlatTree_xlarge_tree                | 🚀 nanogit    | 77.5ms   | 💚 nanogit        | 10.2 MB      |
| UpdateFile_large_repo                  | 🚀 nanogit    | 63.0ms   | 💚 nanogit        | 3.2 MB       |
| UpdateFile_medium_repo                 | 🚀 nanogit    | 59.8ms   | 💚 nanogit        | 1.4 MB       |
| UpdateFile_small_repo                  | 🚀 nanogit    | 50.9ms   | 💚 nanogit        | 1.4 MB       |
| UpdateFile_xlarge_repo                 | 🚀 nanogit    | 79.2ms   | 💚 nanogit        | 11.3 MB      |

## ⚡ Duration Comparison

| Operation                              | git-cli   | go-git    | nanogit   |
| -------------------------------------- | --------- | --------- | --------- |
| BulkCreateFiles_bulk_1000_files_medium | 10.59s 🐌  | 70.20s 🐌  | 115.7ms 🏆 |
| BulkCreateFiles_bulk_1000_files_small  | 9.73s 🐌   | 18.85s 🐌  | 111.1ms 🏆 |
| BulkCreateFiles_bulk_100_files_medium  | 1.83s 🐌   | 6.30s 🐌   | 83.0ms 🏆  |
| BulkCreateFiles_bulk_100_files_small   | 1.60s 🐌   | 838.9ms 🐌 | 76.8ms 🏆  |
| CompareCommits_adjacent_commits_large  | 1.33s 🐌   | 2.60s 🐌   | 100.1ms 🏆 |
| CompareCommits_adjacent_commits_medium | 732.5ms 🐌 | 409.3ms   | 82.7ms 🏆  |
| CompareCommits_adjacent_commits_small  | 628.3ms 🐌 | 72.4ms ✅  | 67.6ms 🏆  |
| CompareCommits_adjacent_commits_xlarge | 7.65s 🐌   | 20.37s 🐌  | 111.5ms 🏆 |
| CompareCommits_few_commits_large       | 1.40s 🐌   | 2.61s 🐌   | 154.9ms 🏆 |
| CompareCommits_few_commits_medium      | 723.0ms   | 406.9ms   | 157.6ms 🏆 |
| CompareCommits_few_commits_small       | 572.3ms 🐌 | 68.6ms 🏆  | 148.6ms   |
| CompareCommits_few_commits_xlarge      | 7.15s 🐌   | 20.25s 🐌  | 209.4ms 🏆 |
| CompareCommits_max_commits_large       | 1.39s 🐌   | 2.60s 🐌   | 253.5ms 🏆 |
| CompareCommits_max_commits_medium      | 722.5ms   | 406.7ms ✅ | 235.8ms 🏆 |
| CompareCommits_max_commits_small       | 577.3ms 🐌 | 68.3ms 🏆  | 227.6ms   |
| CompareCommits_max_commits_xlarge      | 6.44s 🐌   | 20.30s 🐌  | 336.7ms 🏆 |
| CreateFile_large_repo                  | 2.86s 🐌   | 3.05s 🐌   | 63.5ms 🏆  |
| CreateFile_medium_repo                 | 1.47s 🐌   | 518.7ms 🐌 | 60.6ms 🏆  |
| CreateFile_small_repo                  | 1.31s 🐌   | 106.9ms   | 52.2ms 🏆  |
| CreateFile_xlarge_repo                 | 7.41s 🐌   | 24.71s 🐌  | 80.7ms 🏆  |
| DeleteFile_large_repo                  | 3.01s 🐌   | 3.05s 🐌   | 64.7ms 🏆  |
| DeleteFile_medium_repo                 | 1.51s 🐌   | 517.2ms 🐌 | 55.2ms 🏆  |
| DeleteFile_small_repo                  | 1.33s 🐌   | 107.5ms   | 51.9ms 🏆  |
| DeleteFile_xlarge_repo                 | 6.91s 🐌   | 23.36s 🐌  | 77.4ms 🏆  |
| GetFlatTree_large_tree                 | 1.30s 🐌   | 2.63s 🐌   | 58.4ms 🏆  |
| GetFlatTree_medium_tree                | 682.2ms 🐌 | 450.4ms 🐌 | 53.3ms 🏆  |
| GetFlatTree_small_tree                 | 577.9ms 🐌 | 78.3ms ✅  | 52.9ms 🏆  |
| GetFlatTree_xlarge_tree                | 5.54s 🐌   | 20.03s 🐌  | 77.5ms 🏆  |
| UpdateFile_large_repo                  | 2.98s 🐌   | 3.06s 🐌   | 63.0ms 🏆  |
| UpdateFile_medium_repo                 | 1.49s 🐌   | 513.5ms 🐌 | 59.8ms 🏆  |
| UpdateFile_small_repo                  | 1.31s 🐌   | 104.6ms   | 50.9ms 🏆  |
| UpdateFile_xlarge_repo                 | 7.27s 🐌   | 23.08s 🐌  | 79.2ms 🏆  |

## 💾 Memory Usage Comparison

*Note: git-cli uses disk storage rather than keeping data in memory, so memory comparisons focus on in-memory clients (nanogit vs go-git)*

| Operation                              | git-cli      | go-git     | nanogit    |
| -------------------------------------- | ------------ | ---------- | ---------- |
| BulkCreateFiles_bulk_1000_files_medium | -2902688 B 💾 | 43.1 MB 🔥  | 3.8 MB 🏆   |
| BulkCreateFiles_bulk_1000_files_small  | 1.5 MB 💾     | 15.3 MB    | 3.1 MB 🏆   |
| BulkCreateFiles_bulk_100_files_medium  | 5.1 MB 💾     | 51.6 MB 🔥  | 2.4 MB 🏆   |
| BulkCreateFiles_bulk_100_files_small   | -999952 B 💾  | 9.7 MB 🔥   | 1.7 MB 🏆   |
| CompareCommits_adjacent_commits_large  | 70.5 KB 💾    | 227.7 MB 🔥 | 6.3 MB 🏆   |
| CompareCommits_adjacent_commits_medium | 70.5 KB 💾    | 39.3 MB 🔥  | 2.8 MB 🏆   |
| CompareCommits_adjacent_commits_small  | 70.1 KB 💾    | 5.6 MB     | 1.6 MB 🏆   |
| CompareCommits_adjacent_commits_xlarge | 70.5 KB 💾    | 1.6 GB 🔥   | 16.0 MB 🏆  |
| CompareCommits_few_commits_large       | 71.0 KB 💾    | 228.1 MB 🔥 | 7.3 MB 🏆   |
| CompareCommits_few_commits_medium      | 70.2 KB 💾    | 45.0 MB 🔥  | 2.3 MB 🏆   |
| CompareCommits_few_commits_small       | 70.5 KB 💾    | 6.0 MB     | 2.4 MB 🏆   |
| CompareCommits_few_commits_xlarge      | 70.5 KB 💾    | 1.6 GB 🔥   | 17.5 MB 🏆  |
| CompareCommits_max_commits_large       | 70.5 KB 💾    | 235.7 MB 🔥 | 7.3 MB 🏆   |
| CompareCommits_max_commits_medium      | 71.0 KB 💾    | 40.3 MB 🔥  | 2.6 MB 🏆   |
| CompareCommits_max_commits_small       | 70.5 KB 💾    | 3.2 MB ✅   | 3.0 MB 🏆   |
| CompareCommits_max_commits_xlarge      | 70.5 KB 💾    | 1.6 GB 🔥   | 16.9 MB 🏆  |
| CreateFile_large_repo                  | 135.6 KB 💾   | 293.4 MB 🔥 | 3.5 MB 🏆   |
| CreateFile_medium_repo                 | 136.5 KB 💾   | 36.8 MB 🔥  | 1.5 MB 🏆   |
| CreateFile_small_repo                  | 135.7 KB 💾   | 3.4 MB     | 1.4 MB 🏆   |
| CreateFile_xlarge_repo                 | 135.6 KB 💾   | 2.0 GB 🔥   | 11.2 MB 🏆  |
| DeleteFile_large_repo                  | 135.6 KB 💾   | 281.3 MB 🔥 | 3.4 MB 🏆   |
| DeleteFile_medium_repo                 | 135.6 KB 💾   | 41.0 MB 🔥  | 1.5 MB 🏆   |
| DeleteFile_small_repo                  | 135.9 KB 💾   | 3.1 MB ✅   | 1.6 MB 🏆   |
| DeleteFile_xlarge_repo                 | 136.1 KB 💾   | 2.0 GB 🔥   | 11.4 MB 🏆  |
| GetFlatTree_large_tree                 | 3.2 MB 💾     | 265.7 MB 🔥 | 3.6 MB 🏆   |
| GetFlatTree_medium_tree                | 740.1 KB 💾   | 33.0 MB 🔥  | 1.3 MB 🏆   |
| GetFlatTree_small_tree                 | 154.6 KB 💾   | 4.3 MB 🔥   | 697.8 KB 🏆 |
| GetFlatTree_xlarge_tree                | 18.7 MB 💾    | 1.6 GB 🔥   | 10.2 MB 🏆  |
| UpdateFile_large_repo                  | 135.3 KB 💾   | 280.0 MB 🔥 | 3.2 MB 🏆   |
| UpdateFile_medium_repo                 | 135.5 KB 💾   | 37.2 MB 🔥  | 1.4 MB 🏆   |
| UpdateFile_small_repo                  | 135.9 KB 💾   | 4.5 MB     | 1.4 MB 🏆   |
| UpdateFile_xlarge_repo                 | 135.3 KB 💾   | 2.0 GB 🔥   | 11.3 MB 🏆  |

## 🎯 Nanogit Performance Analysis

### ⚡ Speed Comparison

| Operation                              | vs git-cli     | vs go-git       |
| -------------------------------------- | -------------- | --------------- |
| BulkCreateFiles_bulk_1000_files_medium | 91.5x faster 🚀 | 606.5x faster 🚀 |
| BulkCreateFiles_bulk_1000_files_small  | 87.6x faster 🚀 | 169.6x faster 🚀 |
| BulkCreateFiles_bulk_100_files_medium  | 22.1x faster 🚀 | 76.0x faster 🚀  |
| BulkCreateFiles_bulk_100_files_small   | 20.9x faster 🚀 | 10.9x faster 🚀  |
| CompareCommits_adjacent_commits_large  | 13.3x faster 🚀 | 25.9x faster 🚀  |
| CompareCommits_adjacent_commits_medium | 8.9x faster 🚀  | 4.9x faster 🚀   |
| CompareCommits_adjacent_commits_small  | 9.3x faster 🚀  | ~same ⚖️         |
| CompareCommits_adjacent_commits_xlarge | 68.7x faster 🚀 | 182.7x faster 🚀 |
| CompareCommits_few_commits_large       | 9.1x faster 🚀  | 16.8x faster 🚀  |
| CompareCommits_few_commits_medium      | 4.6x faster 🚀  | 2.6x faster 🚀   |
| CompareCommits_few_commits_small       | 3.9x faster 🚀  | 2.2x slower 🐌   |
| CompareCommits_few_commits_xlarge      | 34.1x faster 🚀 | 96.7x faster 🚀  |
| CompareCommits_max_commits_large       | 5.5x faster 🚀  | 10.3x faster 🚀  |
| CompareCommits_max_commits_medium      | 3.1x faster 🚀  | 1.7x faster ✅   |
| CompareCommits_max_commits_small       | 2.5x faster 🚀  | 3.3x slower 🐌   |
| CompareCommits_max_commits_xlarge      | 19.1x faster 🚀 | 60.3x faster 🚀  |
| CreateFile_large_repo                  | 45.1x faster 🚀 | 48.0x faster 🚀  |
| CreateFile_medium_repo                 | 24.3x faster 🚀 | 8.6x faster 🚀   |
| CreateFile_small_repo                  | 25.1x faster 🚀 | 2.0x faster 🚀   |
| CreateFile_xlarge_repo                 | 91.8x faster 🚀 | 306.2x faster 🚀 |
| DeleteFile_large_repo                  | 46.4x faster 🚀 | 47.2x faster 🚀  |
| DeleteFile_medium_repo                 | 27.3x faster 🚀 | 9.4x faster 🚀   |
| DeleteFile_small_repo                  | 25.6x faster 🚀 | 2.1x faster 🚀   |
| DeleteFile_xlarge_repo                 | 89.3x faster 🚀 | 301.7x faster 🚀 |
| GetFlatTree_large_tree                 | 22.3x faster 🚀 | 45.0x faster 🚀  |
| GetFlatTree_medium_tree                | 12.8x faster 🚀 | 8.5x faster 🚀   |
| GetFlatTree_small_tree                 | 10.9x faster 🚀 | 1.5x faster ✅   |
| GetFlatTree_xlarge_tree                | 71.5x faster 🚀 | 258.3x faster 🚀 |
| UpdateFile_large_repo                  | 47.3x faster 🚀 | 48.5x faster 🚀  |
| UpdateFile_medium_repo                 | 24.9x faster 🚀 | 8.6x faster 🚀   |
| UpdateFile_small_repo                  | 25.7x faster 🚀 | 2.1x faster 🚀   |
| UpdateFile_xlarge_repo                 | 91.8x faster 🚀 | 291.4x faster 🚀 |

### 💾 Memory Comparison

*Note: git-cli uses minimal memory as it stores data on disk, not in memory*

| Operation                              | vs git-cli    | vs go-git     |
| -------------------------------------- | ------------- | ------------- |
| BulkCreateFiles_bulk_1000_files_medium | -1.4x more 💾  | 11.3x less 💚  |
| BulkCreateFiles_bulk_1000_files_small  | 2.1x more 💾   | 5.0x less 💚   |
| BulkCreateFiles_bulk_100_files_medium  | 0.5x more 💾   | 21.8x less 💚  |
| BulkCreateFiles_bulk_100_files_small   | -1.8x more 💾  | 5.8x less 💚   |
| CompareCommits_adjacent_commits_large  | 91.2x more 💾  | 36.3x less 💚  |
| CompareCommits_adjacent_commits_medium | 41.0x more 💾  | 13.9x less 💚  |
| CompareCommits_adjacent_commits_small  | 24.1x more 💾  | 3.4x less 💚   |
| CompareCommits_adjacent_commits_xlarge | 232.3x more 💾 | 102.8x less 💚 |
| CompareCommits_few_commits_large       | 105.0x more 💾 | 31.3x less 💚  |
| CompareCommits_few_commits_medium      | 33.4x more 💾  | 19.6x less 💚  |
| CompareCommits_few_commits_small       | 35.4x more 💾  | 2.5x less 💚   |
| CompareCommits_few_commits_xlarge      | 254.0x more 💾 | 92.6x less 💚  |
| CompareCommits_max_commits_large       | 105.5x more 💾 | 32.4x less 💚  |
| CompareCommits_max_commits_medium      | 37.9x more 💾  | 15.3x less 💚  |
| CompareCommits_max_commits_small       | 43.4x more 💾  | 1.1x less ✅   |
| CompareCommits_max_commits_xlarge      | 245.7x more 💾 | 96.3x less 💚  |
| CreateFile_large_repo                  | 26.2x more 💾  | 84.6x less 💚  |
| CreateFile_medium_repo                 | 11.4x more 💾  | 24.2x less 💚  |
| CreateFile_small_repo                  | 10.8x more 💾  | 2.4x less 💚   |
| CreateFile_xlarge_repo                 | 84.7x more 💾  | 185.8x less 💚 |
| DeleteFile_large_repo                  | 25.9x more 💾  | 82.0x less 💚  |
| DeleteFile_medium_repo                 | 11.0x more 💾  | 28.1x less 💚  |
| DeleteFile_small_repo                  | 11.8x more 💾  | 2.0x less ✅   |
| DeleteFile_xlarge_repo                 | 86.1x more 💾  | 175.0x less 💚 |
| GetFlatTree_large_tree                 | 1.1x more 💾   | 74.7x less 💚  |
| GetFlatTree_medium_tree                | 1.8x more 💾   | 25.6x less 💚  |
| GetFlatTree_small_tree                 | 4.5x more 💾   | 6.3x less 💚   |
| GetFlatTree_xlarge_tree                | 0.5x more 💾   | 160.2x less 💚 |
| UpdateFile_large_repo                  | 24.4x more 💾  | 86.9x less 💚  |
| UpdateFile_medium_repo                 | 10.9x more 💾  | 25.7x less 💚  |
| UpdateFile_small_repo                  | 10.5x more 💾  | 3.2x less 💚   |
| UpdateFile_xlarge_repo                 | 85.7x more 💾  | 178.0x less 💚 |

## 📈 Detailed Statistics

### BulkCreateFiles_bulk_1000_files_medium

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 10.59s       | 10.59s       | -2902688 B | -2902688 B    |
| go-git  | 1    | ⚠️ 0.0%   | 70.20s       | 70.20s       | 43.1 MB    | 43.1 MB       |
| nanogit | 1    | ✅ 100.0% | 115.7ms      | 115.7ms      | 3.8 MB     | 3.8 MB        |

### BulkCreateFiles_bulk_1000_files_small

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 9.73s        | 9.73s        | 1.5 MB     | 1.5 MB        |
| go-git  | 1    | ⚠️ 0.0%   | 18.85s       | 18.85s       | 15.3 MB    | 15.3 MB       |
| nanogit | 1    | ✅ 100.0% | 111.1ms      | 111.1ms      | 3.1 MB     | 3.1 MB        |

### BulkCreateFiles_bulk_100_files_medium

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 1.83s        | 1.83s        | 5.1 MB     | 5.1 MB        |
| go-git  | 1    | ⚠️ 0.0%   | 6.30s        | 6.30s        | 51.6 MB    | 51.6 MB       |
| nanogit | 1    | ✅ 100.0% | 83.0ms       | 83.0ms       | 2.4 MB     | 2.4 MB        |

### BulkCreateFiles_bulk_100_files_small

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 1.60s        | 1.60s        | -999952 B  | -999952 B     |
| go-git  | 1    | ⚠️ 0.0%   | 838.9ms      | 838.9ms      | 9.7 MB     | 9.7 MB        |
| nanogit | 1    | ✅ 100.0% | 76.8ms       | 76.8ms       | 1.7 MB     | 1.7 MB        |

### CompareCommits_adjacent_commits_large

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 1.33s        | 1.33s        | 70.5 KB    | 70.5 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 2.60s        | 2.60s        | 227.7 MB   | 227.7 MB      |
| nanogit | 1    | ✅ 100.0% | 100.1ms      | 100.1ms      | 6.3 MB     | 6.3 MB        |

### CompareCommits_adjacent_commits_medium

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 732.5ms      | 732.5ms      | 70.5 KB    | 70.5 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 409.3ms      | 409.3ms      | 39.3 MB    | 39.3 MB       |
| nanogit | 1    | ✅ 100.0% | 82.7ms       | 82.7ms       | 2.8 MB     | 2.8 MB        |

### CompareCommits_adjacent_commits_small

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 628.3ms      | 628.3ms      | 70.1 KB    | 70.1 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 72.4ms       | 72.4ms       | 5.6 MB     | 5.6 MB        |
| nanogit | 1    | ✅ 100.0% | 67.6ms       | 67.6ms       | 1.6 MB     | 1.6 MB        |

### CompareCommits_adjacent_commits_xlarge

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 7.65s        | 7.65s        | 70.5 KB    | 70.5 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 20.37s       | 20.37s       | 1.6 GB     | 1.6 GB        |
| nanogit | 1    | ✅ 100.0% | 111.5ms      | 111.5ms      | 16.0 MB    | 16.0 MB       |

### CompareCommits_few_commits_large

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 1.40s        | 1.40s        | 71.0 KB    | 71.0 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 2.61s        | 2.61s        | 228.1 MB   | 228.1 MB      |
| nanogit | 1    | ✅ 100.0% | 154.9ms      | 154.9ms      | 7.3 MB     | 7.3 MB        |

### CompareCommits_few_commits_medium

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 723.0ms      | 723.0ms      | 70.2 KB    | 70.2 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 406.9ms      | 406.9ms      | 45.0 MB    | 45.0 MB       |
| nanogit | 1    | ✅ 100.0% | 157.6ms      | 157.6ms      | 2.3 MB     | 2.3 MB        |

### CompareCommits_few_commits_small

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 572.3ms      | 572.3ms      | 70.5 KB    | 70.5 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 68.6ms       | 68.6ms       | 6.0 MB     | 6.0 MB        |
| nanogit | 1    | ✅ 100.0% | 148.6ms      | 148.6ms      | 2.4 MB     | 2.4 MB        |

### CompareCommits_few_commits_xlarge

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 7.15s        | 7.15s        | 70.5 KB    | 70.5 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 20.25s       | 20.25s       | 1.6 GB     | 1.6 GB        |
| nanogit | 1    | ✅ 100.0% | 209.4ms      | 209.4ms      | 17.5 MB    | 17.5 MB       |

### CompareCommits_max_commits_large

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 1.39s        | 1.39s        | 70.5 KB    | 70.5 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 2.60s        | 2.60s        | 235.7 MB   | 235.7 MB      |
| nanogit | 1    | ✅ 100.0% | 253.5ms      | 253.5ms      | 7.3 MB     | 7.3 MB        |

### CompareCommits_max_commits_medium

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 722.5ms      | 722.5ms      | 71.0 KB    | 71.0 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 406.7ms      | 406.7ms      | 40.3 MB    | 40.3 MB       |
| nanogit | 1    | ✅ 100.0% | 235.8ms      | 235.8ms      | 2.6 MB     | 2.6 MB        |

### CompareCommits_max_commits_small

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 577.3ms      | 577.3ms      | 70.5 KB    | 70.5 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 68.3ms       | 68.3ms       | 3.2 MB     | 3.2 MB        |
| nanogit | 1    | ✅ 100.0% | 227.6ms      | 227.6ms      | 3.0 MB     | 3.0 MB        |

### CompareCommits_max_commits_xlarge

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 6.44s        | 6.44s        | 70.5 KB    | 70.5 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 20.30s       | 20.30s       | 1.6 GB     | 1.6 GB        |
| nanogit | 1    | ✅ 100.0% | 336.7ms      | 336.7ms      | 16.9 MB    | 16.9 MB       |

### CreateFile_large_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 2.86s        | 2.93s        | 135.6 KB   | 135.5 KB      |
| go-git  | 3    | ✅ 100.0% | 3.05s        | 3.20s        | 293.4 MB   | 285.0 MB      |
| nanogit | 3    | ✅ 100.0% | 63.5ms       | 67.3ms       | 3.5 MB     | 3.5 MB        |

### CreateFile_medium_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 1.47s        | 1.52s        | 136.5 KB   | 136.8 KB      |
| go-git  | 3    | ✅ 100.0% | 518.7ms      | 534.5ms      | 36.8 MB    | 41.5 MB       |
| nanogit | 3    | ✅ 100.0% | 60.6ms       | 69.3ms       | 1.5 MB     | 1.5 MB        |

### CreateFile_small_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 1.31s        | 1.33s        | 135.7 KB   | 135.6 KB      |
| go-git  | 3    | ✅ 100.0% | 106.9ms      | 110.8ms      | 3.4 MB     | 4.2 MB        |
| nanogit | 3    | ✅ 100.0% | 52.2ms       | 59.1ms       | 1.4 MB     | 1.4 MB        |

### CreateFile_xlarge_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 7.41s        | 8.02s        | 135.6 KB   | 135.5 KB      |
| go-git  | 3    | ✅ 100.0% | 24.71s       | 29.00s       | 2.0 GB     | 2.0 GB        |
| nanogit | 3    | ✅ 100.0% | 80.7ms       | 88.9ms       | 11.2 MB    | 11.3 MB       |

### DeleteFile_large_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 3.01s        | 3.13s        | 135.6 KB   | 135.8 KB      |
| go-git  | 3    | ✅ 100.0% | 3.05s        | 3.17s        | 281.3 MB   | 281.8 MB      |
| nanogit | 3    | ✅ 100.0% | 64.7ms       | 71.4ms       | 3.4 MB     | 3.7 MB        |

### DeleteFile_medium_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 1.51s        | 1.52s        | 135.6 KB   | 135.8 KB      |
| go-git  | 3    | ✅ 100.0% | 517.2ms      | 519.7ms      | 41.0 MB    | 40.4 MB       |
| nanogit | 3    | ✅ 100.0% | 55.2ms       | 58.3ms       | 1.5 MB     | 1.5 MB        |

### DeleteFile_small_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 1.33s        | 1.34s        | 135.9 KB   | 135.8 KB      |
| go-git  | 3    | ✅ 100.0% | 107.5ms      | 109.4ms      | 3.1 MB     | 2.8 MB        |
| nanogit | 3    | ✅ 100.0% | 51.9ms       | 55.6ms       | 1.6 MB     | 1.6 MB        |

### DeleteFile_xlarge_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 6.91s        | 7.17s        | 136.1 KB   | 135.8 KB      |
| go-git  | 3    | ✅ 100.0% | 23.36s       | 25.22s       | 2.0 GB     | 2.0 GB        |
| nanogit | 3    | ✅ 100.0% | 77.4ms       | 79.6ms       | 11.4 MB    | 11.4 MB       |

### GetFlatTree_large_tree

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 1.30s        | 1.30s        | 3.2 MB     | 3.2 MB        |
| go-git  | 1    | ✅ 100.0% | 2.63s        | 2.63s        | 265.7 MB   | 265.7 MB      |
| nanogit | 1    | ✅ 100.0% | 58.4ms       | 58.4ms       | 3.6 MB     | 3.6 MB        |

### GetFlatTree_medium_tree

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 682.2ms      | 682.2ms      | 740.1 KB   | 740.1 KB      |
| go-git  | 1    | ✅ 100.0% | 450.4ms      | 450.4ms      | 33.0 MB    | 33.0 MB       |
| nanogit | 1    | ✅ 100.0% | 53.3ms       | 53.3ms       | 1.3 MB     | 1.3 MB        |

### GetFlatTree_small_tree

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 577.9ms      | 577.9ms      | 154.6 KB   | 154.6 KB      |
| go-git  | 1    | ✅ 100.0% | 78.3ms       | 78.3ms       | 4.3 MB     | 4.3 MB        |
| nanogit | 1    | ✅ 100.0% | 52.9ms       | 52.9ms       | 697.8 KB   | 697.8 KB      |

### GetFlatTree_xlarge_tree

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 5.54s        | 5.54s        | 18.7 MB    | 18.7 MB       |
| go-git  | 1    | ✅ 100.0% | 20.03s       | 20.03s       | 1.6 GB     | 1.6 GB        |
| nanogit | 1    | ✅ 100.0% | 77.5ms       | 77.5ms       | 10.2 MB    | 10.2 MB       |

### UpdateFile_large_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 2.98s        | 3.14s        | 135.3 KB   | 135.1 KB      |
| go-git  | 3    | ✅ 100.0% | 3.06s        | 3.14s        | 280.0 MB   | 282.9 MB      |
| nanogit | 3    | ✅ 100.0% | 63.0ms       | 65.3ms       | 3.2 MB     | 3.5 MB        |

### UpdateFile_medium_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 1.49s        | 1.51s        | 135.5 KB   | 135.1 KB      |
| go-git  | 3    | ✅ 100.0% | 513.5ms      | 516.0ms      | 37.2 MB    | 41.7 MB       |
| nanogit | 3    | ✅ 100.0% | 59.8ms       | 69.3ms       | 1.4 MB     | 1.4 MB        |

### UpdateFile_small_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 1.31s        | 1.34s        | 135.9 KB   | 135.6 KB      |
| go-git  | 3    | ✅ 100.0% | 104.6ms      | 105.8ms      | 4.5 MB     | 4.9 MB        |
| nanogit | 3    | ✅ 100.0% | 50.9ms       | 53.5ms       | 1.4 MB     | 1.4 MB        |

### UpdateFile_xlarge_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 7.27s        | 8.42s        | 135.3 KB   | 135.1 KB      |
| go-git  | 3    | ✅ 100.0% | 23.08s       | 24.56s       | 2.0 GB     | 2.0 GB        |
| nanogit | 3    | ✅ 100.0% | 79.2ms       | 82.6ms       | 11.3 MB    | 11.3 MB       |

