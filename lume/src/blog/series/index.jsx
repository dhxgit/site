export const title = "Post Series";
export const layout = "base.njk";

export default ({ seriesDescriptions }) => {
  const dateOptions = { year: "numeric", month: "2-digit", day: "2-digit" };

  return (
    <>
      <h1 className="text-3xl mb-4">{title}</h1>
      <p className="mb-4">
        TODO: filler text here
      </p>

      <ul class="list-disc ml-4 mb-4">
        {Object
          .keys(seriesDescriptions)
          .map((k) => [k, seriesDescriptions[k]])
          .map(([k, v]) => (
            <li>
              <a href={`/blog/series/${k}`}>{k}</a>: {v}
            </li>
          ))}
      </ul>
    </>
  );
};