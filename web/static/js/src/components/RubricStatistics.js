import React, { useEffect, useState } from 'react';
import { useParams } from "react-router-dom";

export default function RubricStatistics() {
  const highLightRe = /(\/?\s?)([^/]+)$/;
  const [categories, setCategories] = useState(null);
  const [isLoading, setIsLoading] = useState(false);
  const [loadedCategory, setLoadedCategory] = useState(null);
  let { id } = useParams();

  useEffect(() => {
    if (isLoading || loadedCategory === id && categories) {
      return;
    }

    setIsLoading(true);

    fetch(`/api/v1/categories?root_id=${id}`, {
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json'
      }
    }).then(response => response.json())
      .then(categories => {
        setLoadedCategory(id);
        setCategories(categories);
        setIsLoading(false);
      });
  });

  return (
    <table className="table">
      <thead>
        <tr>
          <th>Рубрика</th>
          <th>Кол-во товаров</th>
          <th>Выручка</th>
        </tr>
      </thead>
      <tbody>
        {categories && categories.map(category => {
          return (
            <tr key={category.projectId}>
              <td dangerouslySetInnerHTML={{__html: category.title.replace(highLightRe, "$1<b>$2</b>")}}></td>
              <td>{category.productAmount}</td>
              <td>TODO</td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}
