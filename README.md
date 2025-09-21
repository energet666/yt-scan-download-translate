Сервис сканирования и загрузки из плейлистов YouTube.  

**Установить:**  
[go](https://go.dev) для компиляции  
[vot-cli](https://github.com/FOSWLY/vot-cli) для скачивания перевода  
[ffmpeg](https://ffmpeg.org) для сборки видео контейнера  
[yt-dlp](https://github.com/yt-dlp/yt-dlp) для работы с YouTube  

После первого запуска ```go run .```  создастся директория .private со стандартной конфигурацией ```yt-dlp``` ```yt-dlp.conf```, пустым списком скачанных видео ```downloaded.json``` и списком плейлистов для сканирования ```config.json```. Сразу начнется скачивание из тестового плейлиста. Нужно заполнить ```config.json``` ссылками на свои плейлисты и указать флагом нужен ли автоматический перевод. 

**Пример использования.**  
В своем профиле на YouTube создать плейлист с названием ```download``` и плейлист с названием ```download_translate```. Сделать их доступными по ссылке. Файл ```.private/config.json``` должен иметь следующий вид:  
```json
[
	{
	"playlistURL":"url плейлиста download",
	"translate": false
	},
	{
	"playlistURL":"url плейлиста download_translate",
	"translate": true
	}
]
```  
Теперь видео добавленные в ```download``` будут скачаны, а из ```download_translate``` скачаны и переведены.
